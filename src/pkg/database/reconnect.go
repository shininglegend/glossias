package database

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxRetries     = 3
	retryDelay     = 3 * time.Second
	reconnectDelay = 5 * time.Second
)

// IsConnectionError checks if an error is a connection-related error that warrants reconnection
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "tx closed") ||
		strings.Contains(errStr, "connection closed") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "bad connection") ||
		strings.Contains(errStr, "unexpected eof") ||
		strings.Contains(errStr, "conn busy")
}

// ReconnectableDBTX wraps pgxpool.Pool with SQLC's DBTX interface and automatic reconnection
// This ensures SQLC-generated queries benefit from reconnection logic
type ReconnectableDBTX struct {
	pool      *pgxpool.Pool
	connStr   string
	schemaSQL string
}

// NewReconnectableDBTX creates a new DBTX wrapper with reconnection support
func NewReconnectableDBTX(connStr string, schemaSQL string) (*ReconnectableDBTX, error) {
	dbtx := &ReconnectableDBTX{
		connStr:   connStr,
		schemaSQL: schemaSQL,
	}

	if err := dbtx.reconnect(); err != nil {
		return nil, err
	}

	return dbtx, nil
}

// reconnect establishes a new connection pool
func (d *ReconnectableDBTX) reconnect() error {
	fmt.Println("Attempting to reconnect to the database...")
	config, err := pgxpool.ParseConfig(d.connStr)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Disable prepared statements to avoid cache conflicts
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return fmt.Errorf("failed to ping: %w", err)
	}

	if d.schemaSQL != "" {
		if _, err := pool.Exec(context.Background(), d.schemaSQL); err != nil {
			pool.Close()
			return fmt.Errorf("failed to execute schema: %w", err)
		}
	}

	// Close old pool if exists
	if d.pool != nil {
		d.pool.Close()
	}

	d.pool = pool
	fmt.Println("Database connection pool reconnected successfully")
	return nil
}

// executeWithRetry executes a function with automatic reconnection on connection errors
func (d *ReconnectableDBTX) executeWithRetry(fn func() error) error {
	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}

		if !IsConnectionError(err) {
			return err
		}

		fmt.Printf("Connection error detected (attempt %d/%d): %v\n", attempt, maxRetries, err)

		if attempt < maxRetries {
			time.Sleep(retryDelay)
			if reconnectErr := d.reconnect(); reconnectErr != nil {
				fmt.Printf("Reconnection failed: %v\n", reconnectErr)
				time.Sleep(reconnectDelay)
			}
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", maxRetries, err)
}

// Exec implements DBTX interface with retry logic
func (d *ReconnectableDBTX) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	var tag pgconn.CommandTag
	err := d.executeWithRetry(func() error {
		var execErr error
		tag, execErr = d.pool.Exec(ctx, query, args...)
		return execErr
	})
	return tag, err
}

// Query implements DBTX interface with retry logic
func (d *ReconnectableDBTX) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	var rows pgx.Rows
	err := d.executeWithRetry(func() error {
		var queryErr error
		rows, queryErr = d.pool.Query(ctx, query, args...)
		return queryErr
	})
	return rows, err
}

// retryableRow wraps pgx.Row to add retry logic on Scan
type retryableRow struct {
	dbtx  *ReconnectableDBTX
	ctx   context.Context
	query string
	args  []any
}

func (r *retryableRow) Scan(dest ...any) error {
	var scanErr error
	err := r.dbtx.executeWithRetry(func() error {
		row := r.dbtx.pool.QueryRow(r.ctx, r.query, r.args...)
		scanErr = row.Scan(dest...)
		return scanErr
	})
	return err
}

// QueryRow implements DBTX interface with retry logic via retryableRow
func (d *ReconnectableDBTX) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return &retryableRow{
		dbtx:  d,
		ctx:   ctx,
		query: query,
		args:  args,
	}
}

// CopyFrom implements DBTX interface with retry logic
func (d *ReconnectableDBTX) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	var rowsAffected int64
	err := d.executeWithRetry(func() error {
		var copyErr error
		rowsAffected, copyErr = d.pool.CopyFrom(ctx, tableName, columnNames, rowSrc)
		return copyErr
	})
	return rowsAffected, err
}

// Close closes the connection pool
func (d *ReconnectableDBTX) Close() {
	if d.pool != nil {
		d.pool.Close()
	}
}

// Pool returns the underlying pool (for any direct access needs)
func (d *ReconnectableDBTX) Pool() *pgxpool.Pool {
	return d.pool
}

// InitDBWithReconnect initializes database with automatic reconnection support
func InitDBWithReconnect(dbPath string) (Store, error) {
	usePool := os.Getenv("USE_POOL") != "false"

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		return &mockStore{}, nil
	}

	// Read schema
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return nil, err
	}

	if usePool {
		// Create reconnectable DBTX wrapper for SQLC compatibility
		dbtx, err := NewReconnectableDBTX(connStr, string(schema))
		if err != nil {
			return nil, err
		}
		return &reconnectableDBTXStore{dbtx: dbtx}, nil
	}

	// Fallback to regular connection for non-pool mode
	return InitDB(dbPath)
}

// reconnectableDBTXStore wraps ReconnectableDBTX as a Store
type reconnectableDBTXStore struct {
	dbtx *ReconnectableDBTX
}

func (s *reconnectableDBTXStore) DB() DB {
	// Not typically used - SQLC queries use RawConn() directly
	// Return nil to indicate this store doesn't support the legacy DB interface
	return nil
}

func (s *reconnectableDBTXStore) Close() error {
	s.dbtx.Close()
	return nil
}

func (s *reconnectableDBTXStore) RawConn() any {
	// Return the DBTX wrapper, not the raw pool
	// This is what gets passed to db.New() in models.SetDB()
	return s.dbtx
}
