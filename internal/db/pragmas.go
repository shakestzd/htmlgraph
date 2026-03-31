// Package db provides SQLite database operations for HtmlGraph.
//
// Uses modernc.org/sqlite (pure Go, no CGo) for maximum portability.
package db

import (
	"database/sql"
	"fmt"
)

// Pragmas mirrors the Python PRAGMA_SETTINGS from pragmas.py.
var Pragmas = map[string]string{
	"journal_mode": "WAL",
	"synchronous":  "NORMAL",
	"foreign_keys": "1",
	"busy_timeout": "5000",
	"cache_size":   "-64000",
	"temp_store":   "MEMORY",
	"mmap_size":    "268435456",
}

// ApplyPragmas sets all performance PRAGMAs on a database connection.
func ApplyPragmas(db *sql.DB) error {
	for pragma, value := range Pragmas {
		_, err := db.Exec(fmt.Sprintf("PRAGMA %s = %s", pragma, value))
		if err != nil {
			return fmt.Errorf("applying PRAGMA %s: %w", pragma, err)
		}
	}
	return nil
}

// RunOptimize executes PRAGMA optimize for planner/statistics upkeep.
func RunOptimize(db *sql.DB) error {
	_, err := db.Exec("PRAGMA optimize")
	return err
}

// CheckIntegrity runs integrity_check and foreign_key_check.
// Returns true if the database passes both checks.
func CheckIntegrity(db *sql.DB) (bool, error) {
	row := db.QueryRow("PRAGMA integrity_check")
	var result string
	if err := row.Scan(&result); err != nil {
		return false, fmt.Errorf("integrity_check: %w", err)
	}
	if result != "ok" {
		return false, nil
	}

	rows, err := db.Query("PRAGMA foreign_key_check")
	if err != nil {
		return false, fmt.Errorf("foreign_key_check: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		// Any row means a violation exists.
		return false, nil
	}
	return true, rows.Err()
}
