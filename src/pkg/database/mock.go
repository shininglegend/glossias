// logos-stories/src/pkg/database/mock.go
package database

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
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

// MockQueryResult stores the expected return values for a query
type MockQueryResult struct {
	Rows [][]interface{}
	Err  error
}

// MockDBTX implements db.DBTX for testing with query stubbing
type MockDBTX struct {
	queries map[string]MockQueryResult
	execs   map[string]error
}

// NewMockDBTX creates a new mock DBTX connection with query stubbing capabilities
func NewMockDBTX() *MockDBTX {
	return &MockDBTX{
		queries: make(map[string]MockQueryResult),
		execs:   make(map[string]error),
	}
}

// StubQuery registers a mocked result for any query containing the given substring
func (m *MockDBTX) StubQuery(querySubstr string, rows [][]interface{}, err error) {
	m.queries[querySubstr] = MockQueryResult{Rows: rows, Err: err}
}

// StubExec registers a mocked error (or nil for success) for exec queries matching substring
func (m *MockDBTX) StubExec(querySubstr string, err error) {
	m.execs[querySubstr] = err
}

func (m *MockDBTX) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	for k, err := range m.execs {
		if strings.Contains(sql, k) {
			return pgconn.CommandTag{}, err
		}
	}
	return pgconn.CommandTag{}, nil
}

func (m *MockDBTX) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	for k, qr := range m.queries {
		if strings.Contains(sql, k) {
			if qr.Err != nil {
				return nil, qr.Err
			}
			return &MockPgxRows{rows: qr.Rows}, nil
		}
	}
	return &MockPgxRows{}, nil
}

func (m *MockDBTX) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	for k, qr := range m.queries {
		if strings.Contains(sql, k) {
			if qr.Err != nil {
				return &MockPgxRow{err: qr.Err}
			}
			if len(qr.Rows) > 0 {
				return &MockPgxRow{row: qr.Rows[0]}
			}
			return &MockPgxRow{err: pgx.ErrNoRows}
		}
	}
	return &MockPgxRow{err: pgx.ErrNoRows}
}

func (m *MockDBTX) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

// Mock implementations for pgx types
type MockPgxRows struct {
	rows [][]interface{}
	curr int
	err  error
}

func (m *MockPgxRows) Next() bool {
	if m.err != nil || m.curr >= len(m.rows) {
		return false
	}
	return true
}

func (m *MockPgxRows) Scan(dest ...interface{}) error {
	if m.err != nil {
		return m.err
	}
	if m.curr >= len(m.rows) {
		return pgx.ErrNoRows
	}
	err := scanValues(m.rows[m.curr], dest)
	if err != nil {
		return err
	}
	m.curr++
	return nil
}

func (m *MockPgxRows) Values() ([]interface{}, error)               { return nil, nil }
func (m *MockPgxRows) Close()                                       {}
func (m *MockPgxRows) Err() error                                   { return m.err }
func (m *MockPgxRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (m *MockPgxRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (m *MockPgxRows) RawValues() [][]byte                          { return nil }
func (m *MockPgxRows) Conn() *pgx.Conn                              { return nil }

type MockPgxRow struct {
	row []interface{}
	err error
}

func (m *MockPgxRow) Scan(dest ...interface{}) error {
	if m.err != nil {
		return m.err
	}
	if len(m.row) == 0 {
		return pgx.ErrNoRows
	}
	return scanValues(m.row, dest)
}

func scanValues(src []interface{}, dest []interface{}) error {
	if len(src) != len(dest) {
		return fmt.Errorf("column count mismatch: src %d, dest %d", len(src), len(dest))
	}
	for i, val := range src {
		d := reflect.ValueOf(dest[i])
		if d.Kind() != reflect.Ptr {
			return fmt.Errorf("dest element %d is not a pointer", i)
		}
		if d.IsNil() {
			return fmt.Errorf("dest element %d is nil pointer", i)
		}

		if val == nil {
			d.Elem().Set(reflect.Zero(d.Elem().Type()))
			continue
		}

		sVal := reflect.ValueOf(val)
		if sVal.Type().AssignableTo(d.Elem().Type()) {
			d.Elem().Set(sVal)
		} else if sVal.Type().ConvertibleTo(d.Elem().Type()) {
			d.Elem().Set(sVal.Convert(d.Elem().Type()))
		} else {
			return fmt.Errorf("cannot assign %s to %s at index %d", sVal.Type(), d.Elem().Type(), i)
		}
	}
	return nil
}
