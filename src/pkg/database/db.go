// logos-stories/src/pkg/database/db.go
package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaFS embed.FS

// InitDB selects PostgreSQL implementation based on USE_POOL environment variable
// USE_POOL=true uses pgxpool, USE_POOL=false uses database/sql
func InitDB(dbPath string) (Store, error) {
	usePool, _ := strconv.ParseBool(os.Getenv("USE_POOL"))

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		return &mockStore{}, nil
	}

	// Initialize schema
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return nil, err
	}

	if usePool {
		// Use pgxpool for PostgreSQL
		pool, err := pgxpool.New(context.Background(), connStr)
		if err != nil {
			return nil, err
		}

		if err := pool.Ping(context.Background()); err != nil {
			return nil, err
		}

		if _, err := pool.Exec(context.Background(), string(schema)); err != nil {
			return nil, err
		}

		return &poolStore{pool: pool}, nil
	}

	// Use database/sql with postgres driver
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if _, err := db.Exec(string(schema)); err != nil {
		return nil, err
	}

	return &sqlStore{db: &sqlDB{db}}, nil
}

// mockStore for when no DATABASE_URL is provided
type mockStore struct{}

func (s *mockStore) DB() DB {
	return &mockDB{}
}

func (s *mockStore) Close() error {
	return nil
}

// mockDB implements DB interface with no-ops
type mockDB struct{}

func (db *mockDB) Exec(query string, args ...any) (sql.Result, error) {
	return &mockResult{}, nil
}

func (db *mockDB) Query(query string, args ...any) (Rows, error) {
	return &mockRows{}, nil
}

func (db *mockDB) QueryRow(query string, args ...any) Row {
	return &mockRow{}
}

func (db *mockDB) Begin() (*sql.Tx, error) {
	return nil, errors.New("transactions not supported in mock")
}

func (db *mockDB) Close() error {
	return nil
}

func (db *mockDB) Ping() error {
	return nil
}

// mockResult implements sql.Result
type mockResult struct{}

func (r *mockResult) LastInsertId() (int64, error) {
	return 1, nil
}

func (r *mockResult) RowsAffected() (int64, error) {
	return 1, nil
}

// mockRows implements Rows interface
type mockRows struct{}

func (r *mockRows) Close() error {
	return nil
}

func (r *mockRows) Next() bool {
	return false
}

func (r *mockRows) Scan(dest ...any) error {
	return nil
}

func (r *mockRows) Columns() ([]string, error) {
	return []string{}, nil
}

// mockRow implements Row interface
type mockRow struct{}

func (r *mockRow) Scan(dest ...any) error {
	return sql.ErrNoRows
}

// poolStore wraps pgxpool.Pool to implement Store interface
type poolStore struct {
	pool *pgxpool.Pool
}

func (s *poolStore) DB() DB {
	return &poolDB{s.pool}
}

func (s *poolStore) Close() error {
	s.pool.Close()
	return nil
}

// poolDB wraps pgxpool.Pool to implement DB interface
type poolDB struct {
	pool *pgxpool.Pool
}

func (db *poolDB) Exec(query string, args ...any) (sql.Result, error) {
	tag, err := db.pool.Exec(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	return &poolResult{tag}, nil
}

func (db *poolDB) Query(query string, args ...any) (Rows, error) {
	rows, err := db.pool.Query(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	return &poolRows{rows}, nil
}

func (db *poolDB) QueryRow(query string, args ...any) Row {
	return &poolRow{db.pool.QueryRow(context.Background(), query, args...)}
}

func (db *poolDB) Begin() (*sql.Tx, error) {
	return nil, errors.New("transactions not implemented for pgxpool")
}

func (db *poolDB) Close() error {
	db.pool.Close()
	return nil
}

func (db *poolDB) Ping() error {
	return db.pool.Ping(context.Background())
}

// poolResult wraps pgx CommandTag
type poolResult struct {
	tag any
}

func (r *poolResult) LastInsertId() (int64, error) {
	return 0, errors.New("LastInsertId not supported by pgx")
}

func (r *poolResult) RowsAffected() (int64, error) {
	if tag, ok := r.tag.(interface{ RowsAffected() int64 }); ok {
		return tag.RowsAffected(), nil
	}
	return 0, nil
}

// poolRows wraps pgx.Rows
type poolRows struct {
	rows any
}

func (r *poolRows) Close() error {
	if rows, ok := r.rows.(interface{ Close() }); ok {
		rows.Close()
	}
	return nil
}

func (r *poolRows) Next() bool {
	if rows, ok := r.rows.(interface{ Next() bool }); ok {
		return rows.Next()
	}
	return false
}

func (r *poolRows) Scan(dest ...any) error {
	if rows, ok := r.rows.(interface{ Scan(dest ...any) error }); ok {
		return rows.Scan(dest...)
	}
	return errors.New("scan not supported")
}

func (r *poolRows) Columns() ([]string, error) {
	return nil, errors.New("columns not implemented for pgx")
}

// poolRow wraps pgx.Row
type poolRow struct {
	row any
}

func (r *poolRow) Scan(dest ...any) error {
	if row, ok := r.row.(interface{ Scan(dest ...any) error }); ok {
		return row.Scan(dest...)
	}
	return errors.New("scan not supported")
}

// sqlStore wraps database/sql DB to implement Store interface
type sqlStore struct {
	db DB
}

func (s *sqlStore) DB() DB {
	return s.db
}

func (s *sqlStore) Close() error {
	return s.db.Close()
}

// sqlDB wraps sql.DB to implement DB interface
type sqlDB struct {
	*sql.DB
}

// sqlRows wraps sql.Rows
type sqlRows struct {
	*sql.Rows
}

// sqlRow wraps sql.Row
type sqlRow struct {
	*sql.Row
}

func (db *sqlDB) Query(query string, args ...any) (Rows, error) {
	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRows{rows}, nil
}

func (db *sqlDB) QueryRow(query string, args ...any) Row {
	return &sqlRow{db.DB.QueryRow(query, args...)}
}
