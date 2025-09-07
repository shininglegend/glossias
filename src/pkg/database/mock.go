// logos-stories/src/pkg/database/mock.go
package database

import (
	"context"
	"database/sql"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// MockDB implements the DB interface for testing
type MockDB struct {
	mu   sync.RWMutex
	data map[string][]interface{} // Basic in-memory storage
	tx   *MockTx
}

type MockTx struct {
	db *MockDB
}

type MockResult struct {
	lastID  int64
	rowsAff int64
}

type MockRows struct {
	closed bool
	rows   [][]interface{}
	curr   int
}

type MockRow struct {
	data []interface{}
}

func (r MockResult) LastInsertId() (int64, error) { return r.lastID, nil }
func (r MockResult) RowsAffected() (int64, error) { return r.rowsAff, nil }

// NewMockDB creates a new mock database
func NewMockDB() *MockDB {
	return &MockDB{
		data: make(map[string][]interface{}),
	}
}

func (m *MockDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simple implementation - just store the args with the query as key
	m.data[query] = args
	return MockResult{1, 1}, nil
}

func (m *MockDB) Begin() (*sql.Tx, error) {
	m.tx = &MockTx{db: m}
	return &sql.Tx{}, nil
}

func (m *MockDB) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = nil
	return nil
}

func (m *MockDB) Ping() error {
	return nil
}

func (m *MockRows) Close() error {
	m.closed = true
	return nil
}

func (m *MockRows) Next() bool {
	if m.closed || m.curr >= len(m.rows) {
		return false
	}
	m.curr++
	return true
}

func (m *MockRows) Scan(dest ...interface{}) error {
	return sql.ErrNoRows
}

func (m *MockRows) Columns() ([]string, error) {
	return []string{}, nil
}

func (m *MockRow) Scan(dest ...interface{}) error {
	return sql.ErrNoRows
}

func (m *MockDB) Query(query string, args ...interface{}) (Rows, error) {
	return &MockRows{}, nil
}

func (m *MockDB) QueryRow(query string, args ...interface{}) Row {
	return &MockRow{}
}

// MockDBTX implements db.DBTX for testing
type MockDBTX struct{}

func (m *MockDBTX) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (m *MockDBTX) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return &MockPgxRows{}, nil
}

func (m *MockDBTX) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return &MockPgxRow{}
}

func (m *MockDBTX) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

// Mock implementations for pgx types
type MockPgxRows struct{}

func (m *MockPgxRows) Next() bool                                   { return false }
func (m *MockPgxRows) Scan(dest ...interface{}) error               { return nil }
func (m *MockPgxRows) Values() ([]interface{}, error)               { return nil, nil }
func (m *MockPgxRows) Close()                                       {}
func (m *MockPgxRows) Err() error                                   { return nil }
func (m *MockPgxRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (m *MockPgxRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (m *MockPgxRows) RawValues() [][]byte                          { return nil }
func (m *MockPgxRows) Conn() *pgx.Conn                              { return nil }

type MockPgxRow struct{}

func (m *MockPgxRow) Scan(dest ...interface{}) error { return nil }
