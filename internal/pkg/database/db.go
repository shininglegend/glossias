// logos-stories/internal/pkg/database/db.go
package database

import (
	"database/sql"
	"embed"
	"os"

	_ "github.com/lib/pq"
)

//go:embed schema.sql
var schemaFS embed.FS

// realDB wraps sql.DB to implement our DB interface
type realDB struct {
	*sql.DB
}

// RealRows wraps sql.Rows
type RealRows struct {
	*sql.Rows
}

// RealRow wraps sql.Row
type RealRow struct {
	*sql.Row
}

// InitDB creates and initializes the database
func InitDB(dbPath string) (Store, error) {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		// Use mock DB for testing
		return &mockStore{db: NewMockDB()}, nil
	}

	// Use real PostgreSQL database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Initialize schema
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(string(schema)); err != nil {
		return nil, err
	}

	return &realStore{db: &realDB{db}}, nil
}

type realStore struct {
	db DB
}

func (s *realStore) DB() DB {
	return s.db
}

func (s *realStore) Close() error {
	return s.db.Close()
}

type mockStore struct {
	db DB
}

func (s *mockStore) DB() DB {
	return s.db
}

func (s *mockStore) Close() error {
	return s.db.Close()
}

func (db *realDB) Query(query string, args ...interface{}) (Rows, error) {
	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	return &RealRows{rows}, nil
}

func (db *realDB) QueryRow(query string, args ...interface{}) Row {
	return &RealRow{db.DB.QueryRow(query, args...)}
}
