//go:build integration

// Package testdb runs integration tests against a real Postgres: one
// container per test process, migrated with the production goose migrations
// and seeded once into a template database. Read-only tests share a clone of
// the template; tests that write get their own clone (CREATE DATABASE ...
// TEMPLATE), so withTransaction opens real transactions exactly as in
// production and nothing leaks between tests.
//
// Build with -tags integration. Without Docker the tests skip. TESTDB_URL
// points the harness at an existing server instead of a container; it must be
// a throwaway server where the user may CREATE DATABASE, never the .env
// DATABASE_URL.
package testdb

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"glossias/src/pkg/database"
	"glossias/src/pkg/models"
	"glossias/src/pkg/testdb/seed"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// Fixture is a database the models package is pointed at.
type Fixture struct {
	Pool *pgxpool.Pool
	Name string
	Data *seed.Dataset
}

var (
	once     sync.Once
	startErr error

	adminURL  *url.URL // server URL; the path selects the database
	adminPool *pgxpool.Pool
	container *tcpostgres.PostgresContainer
	suffix    string
	dataset   *seed.Dataset
	shared    *Fixture
	clones    atomic.Int32
)

func templateName() string { return "glossias_tmpl" + suffix }

// Spec is the dataset size: seed.Realistic, or seed.Small with
// INTEGRATION_SPEC=small.
func Spec() seed.Spec {
	s := seed.Realistic
	if os.Getenv("INTEGRATION_SPEC") == "small" {
		s = seed.Small
	}
	// Entitlement expiry is compared with the database clock.
	s.Now = time.Now()
	return s
}

// Shared points models at the shared read-only database and returns it.
// Tests using it must not write. It skips t when Docker is unavailable.
func Shared(t *testing.T) *Fixture {
	t.Helper()
	start(t)
	models.SetDB(shared.Pool)
	return shared
}

// Fresh clones the seeded template into a new database for one test, points
// models at it and clears the cache. Cleanup restores the shared database and
// drops the clone.
func Fresh(t *testing.T) *Fixture {
	t.Helper()
	start(t)
	ctx := context.Background()
	name := fmt.Sprintf("glossias_t%04d%s", clones.Add(1), suffix)
	pool, err := clone(ctx, name)
	if err != nil {
		t.Fatalf("clone test database: %v", err)
	}
	models.SetDB(pool)
	models.ClearAllCache()
	t.Cleanup(func() {
		models.SetDB(shared.Pool)
		models.ClearAllCache()
		pool.Close()
		if _, err := adminPool.Exec(ctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)"); err != nil {
			t.Logf("drop %s: %v", name, err)
		}
	})
	return &Fixture{Pool: pool, Name: name, Data: dataset}
}

// Terminate stops the container (or drops the databases created on a
// TESTDB_URL server). Call it from TestMain after m.Run.
func Terminate() {
	ctx := context.Background()
	if shared != nil {
		shared.Pool.Close()
	}
	if container != nil {
		_ = container.Terminate(ctx)
		return
	}
	if adminPool != nil {
		for _, name := range []string{shared.Name, templateName()} {
			_, _ = adminPool.Exec(ctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)")
		}
		adminPool.Close()
	}
}

func start(t *testing.T) {
	t.Helper()
	if os.Getenv("TESTDB_URL") == "" {
		testcontainers.SkipIfProviderIsNotHealthy(t)
	}
	once.Do(func() { startErr = setup() })
	if startErr != nil {
		t.Fatalf("integration database: %v", startErr)
	}
}

func setup() error {
	ctx := context.Background()
	raw := os.Getenv("TESTDB_URL")
	if raw != "" {
		if raw == os.Getenv("DATABASE_URL") {
			return fmt.Errorf("TESTDB_URL must not be DATABASE_URL: the harness creates and drops databases")
		}
		// Several runs may share the server.
		suffix = fmt.Sprintf("_%d", os.Getpid())
	} else {
		c, err := tcpostgres.Run(ctx, "postgres:17-alpine",
			tcpostgres.WithDatabase("postgres"),
			tcpostgres.WithUsername("postgres"),
			tcpostgres.WithPassword("postgres"),
			testcontainers.WithCmd("postgres", "-c", "fsync=off", "-c", "synchronous_commit=off", "-c", "full_page_writes=off"),
			tcpostgres.BasicWaitStrategies(),
		)
		container = c
		if err != nil {
			return fmt.Errorf("start postgres container: %w", err)
		}
		raw, err = c.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			return err
		}
	}
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	adminURL = u
	adminPool, err = newPool(ctx, dbURL("postgres"))
	if err != nil {
		return fmt.Errorf("connect to server: %w", err)
	}

	tmpl := templateName()
	if _, err := adminPool.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{tmpl}.Sanitize()); err != nil {
		return fmt.Errorf("create template: %w", err)
	}
	if err := database.RunMigrations(dbURL(tmpl)); err != nil {
		return fmt.Errorf("migrate template: %w", err)
	}
	tp, err := newPool(ctx, dbURL(tmpl))
	if err != nil {
		return err
	}
	dataset = seed.Generate(Spec())
	if err := seed.Load(ctx, tp, dataset); err != nil {
		tp.Close()
		return err
	}
	if _, err := tp.Exec(ctx, "VACUUM ANALYZE"); err != nil {
		tp.Close()
		return err
	}
	// CREATE DATABASE ... TEMPLATE needs the template to have no sessions.
	tp.Close()

	name := "glossias_ro" + suffix
	pool, err := clone(ctx, name)
	if err != nil {
		return err
	}
	shared = &Fixture{Pool: pool, Name: name, Data: dataset}
	return nil
}

func clone(ctx context.Context, name string) (*pgxpool.Pool, error) {
	q := fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s",
		pgx.Identifier{name}.Sanitize(), pgx.Identifier{templateName()}.Sanitize())
	if _, err := adminPool.Exec(ctx, q); err != nil {
		return nil, fmt.Errorf("create %s: %w", name, err)
	}
	return newPool(ctx, dbURL(name))
}

func dbURL(name string) string {
	u := *adminURL
	u.Path = "/" + name
	return u.String()
}

// newPool mirrors database.InitDB: simple protocol, as production uses.
func newPool(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
