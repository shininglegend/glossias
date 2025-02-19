// logos-stories/internal/pkg/database/interface.go
package database

import (
	"database/sql"
)

// DB represents our database operations interface
type DB interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (Rows, error)
	QueryRow(query string, args ...interface{}) Row
	Begin() (*sql.Tx, error)
	Close() error
	Ping() error
}

// Store represents our high-level storage interface
type Store interface {
	DB() DB
	Close() error
}

type Rows interface {
	Close() error
	Next() bool
	Scan(dest ...interface{}) error
	Columns() ([]string, error)
}

type Row interface {
	Scan(dest ...interface{}) error
}
