// logos-stories/internal/pkg/database/db.go
package database

import (
	"context"
	"embed"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaFS embed.FS

// InitDB creates and initializes the database
func InitDB(dbPath string) (*pgxpool.Pool, error) {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		return nil, nil // Return nil for mock mode
	}

	// Use pgxpool for PostgreSQL database
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, err
	}

	// Initialize schema
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return nil, err
	}

	if _, err := pool.Exec(context.Background(), string(schema)); err != nil {
		return nil, err
	}

	return pool, nil
}

// Legacy compatibility - keeping minimal interface for existing code
type mockStore struct {
	db DB
}

func (s *mockStore) DB() DB {
	return s.db
}

func (s *mockStore) Close() error {
	return s.db.Close()
}
