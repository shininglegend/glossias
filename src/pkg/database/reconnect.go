package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxRetries     = 3
	retryDelay     = 1 * time.Second
	reconnectDelay = 2 * time.Second
)

// isConnectionError checks if an error is a connection-related error that warrants reconnection
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "tx closed") ||
		strings.Contains(errStr, "connection closed") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "bad connection") ||
		strings.Contains(errStr, "unexpected eof") ||
		strings.Contains(errStr, "conn busy")
}

// ReconnectablePoolStore wraps poolStore with automatic reconnection
type ReconnectablePoolStore struct {
	pool      *pgxpool.Pool
	connStr   string
	schemaSQL string
}

// NewReconnectablePoolStore creates a new reconnectable pool store
func NewReconnectablePoolStore(connStr string, schemaSQL string) (*ReconnectablePoolStore, error) {
	store := &ReconnectablePoolStore{
		connStr:   connStr,
		schemaSQL: schemaSQL,
	}

	if err := store.reconnect(); err != nil {
		return nil, err
	}

	return store, nil
}

// reconnect establishes a new connection pool
func (s *ReconnectablePoolStore) reconnect() error {
	config, err := pgxpool.ParseConfig(s.connStr)
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

	if s.schemaSQL != "" {
		if _, err := pool.Exec(context.Background(), s.schemaSQL); err != nil {
			pool.Close()
			return fmt.Errorf("failed to execute schema: %w", err)
		}
	}

	// Close old pool if exists
	if s.pool != nil {
		s.pool.Close()
	}

	s.pool = pool
	fmt.Println("Database connection pool reconnected successfully")
	return nil
}

// executeWithRetry executes a function with automatic reconnection on connection errors
func (s *ReconnectablePoolStore) executeWithRetry(fn func() error) error {
	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}

		if !isConnectionError(err) {
			return err
		}

		fmt.Printf("Connection error detected (attempt %d/%d): %v\n", attempt, maxRetries, err)

		if attempt < maxRetries {
			time.Sleep(retryDelay)
			if reconnectErr := s.reconnect(); reconnectErr != nil {
				fmt.Printf("Reconnection failed: %v\n", reconnectErr)
				time.Sleep(reconnectDelay)
			}
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", maxRetries, err)
}

func (s *ReconnectablePoolStore) DB() DB {
	return &reconnectablePoolDB{store: s}
}

func (s *ReconnectablePoolStore) Close() error {
	if s.pool != nil {
		s.pool.Close()
	}
	return nil
}

func (s *ReconnectablePoolStore) RawConn() any {
	return s.pool
}

// reconnectablePoolDB wraps pool operations with retry logic
type reconnectablePoolDB struct {
	store *ReconnectablePoolStore
}

func (db *reconnectablePoolDB) Exec(query string, args ...any) (sql.Result, error) {
	var result sql.Result
	err := db.store.executeWithRetry(func() error {
		tag, execErr := db.store.pool.Exec(context.Background(), query, args...)
		if execErr != nil {
			return execErr
		}
		result = &poolResult{tag}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (db *reconnectablePoolDB) Query(query string, args ...any) (result Rows, err error) {
	err = db.store.executeWithRetry(func() error {
		rows, queryErr := db.store.pool.Query(context.Background(), query, args...)
		if queryErr != nil {
			return queryErr
		}
		result = &poolRows{rows}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (db *reconnectablePoolDB) QueryRow(query string, args ...any) Row {
	// QueryRow doesn't return errors until Scan is called
	return &poolRow{db.store.pool.QueryRow(context.Background(), query, args...)}
}

func (db *reconnectablePoolDB) Begin() (*sql.Tx, error) {
	return nil, fmt.Errorf("transactions not implemented for reconnectable pool")
}

func (db *reconnectablePoolDB) Close() error {
	return db.store.Close()
}

func (db *reconnectablePoolDB) Ping() error {
	return db.store.executeWithRetry(func() error {
		return db.store.pool.Ping(context.Background())
	})
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
		return NewReconnectablePoolStore(connStr, string(schema))
	}

	// Fallback to regular connection for non-pool mode
	return InitDB(dbPath)
}
