package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// sql.Row re-export so callers don't need to import database/sql directly.
type Row = sql.Row

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

func (db *DB) Close() error { return db.conn.Close() }

// QueryRow exposes a single-row query for health checks and small lookups.
func (db *DB) QueryRow(query string, args ...any) *sql.Row {
	return db.conn.QueryRow(query, args...)
}
