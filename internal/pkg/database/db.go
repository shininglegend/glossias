// logos-stories/internal/pkg/database/db.go
package database

import (
	"database/sql"
	"embed"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaFS embed.FS

// InitDB creates and initializes the database
func InitDB(dbPath string) (*sql.DB, error) {
	// Check if we need to create a new database
	needsInit := false
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		needsInit = true
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, err
	}

	// Initialize schema if needed
	if needsInit {
		schema, err := schemaFS.ReadFile("schema.sql")
		if err != nil {
			return nil, err
		}

		if _, err := db.Exec(string(schema)); err != nil {
			return nil, err
		}
	}

	return db, nil
}
