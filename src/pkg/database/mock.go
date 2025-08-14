// logos-stories/src/pkg/database/mock.go
package database

import (
	"database/sql"
	"sync"
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
