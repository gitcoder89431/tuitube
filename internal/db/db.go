package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("pragma: %w", err)
	}
	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// Migrate adds any missing columns/tables introduced after the initial schema.
func (db *DB) Migrate() error {
	// tags column — added for Last.fm enrichment
	_, err := db.conn.Exec(`ALTER TABLE tracks ADD COLUMN tags TEXT`)
	if err != nil && err.Error() != "duplicate column name: tags" {
		// ignore "already exists" errors, surface real ones
		if !isAlreadyExists(err) {
			return err
		}
	}
	return nil
}

func isAlreadyExists(err error) bool {
	s := err.Error()
	return contains(s, "duplicate column") || contains(s, "already exists")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
