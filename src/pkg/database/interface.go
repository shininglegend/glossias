// logos-stories/src/pkg/database/interface.go
package database

import (
	"database/sql"
)

// DB represents our database operations interface
type DB interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (Rows, error)
	QueryRow(query string, args ...any) Row
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
	Scan(dest ...any) error
	Columns() ([]string, error)
}

type Row interface {
	Scan(dest ...any) error
}
