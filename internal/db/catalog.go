package db

import (
	"fmt"
	"os"
	"path/filepath"
)

// CatalogVersion is stored in both catalog.db and the user DB to detect upgrades.
const catalogVersionKey = "catalog_version"

// InitUserDB ensures the user DB has all required tables including the meta
// table used to track which catalog version was last merged.
func (db *DB) InitUserDB() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS meta (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS playlists (
			id         INTEGER PRIMARY KEY,
			name       TEXT NOT NULL,
			created_at INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000)
		);
		CREATE TABLE IF NOT EXISTS playlist_tracks (
			playlist_id INTEGER NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
			track_id    TEXT NOT NULL,
			added_at    INTEGER NOT NULL DEFAULT (strftime('%s','now') * 1000),
			PRIMARY KEY (playlist_id, track_id)
		);
		CREATE TABLE IF NOT EXISTS downloads (
			youtube_id    TEXT PRIMARY KEY,
			filepath      TEXT NOT NULL,
			downloaded_at INTEGER NOT NULL DEFAULT (strftime('%s','now')*1000)
		);
		INSERT OR IGNORE INTO playlists (id, name) VALUES (1, 'Favorites');
	`)
	return err
}

// CatalogVersion returns the catalog version string stored in the meta table.
// Returns "" if not set.
func (db *DB) CatalogVersion() string {
	var v string
	db.conn.QueryRow("SELECT value FROM meta WHERE key=?", catalogVersionKey).Scan(&v)
	return v
}

// MergeCatalog attaches catalogPath and merges any new stations and tracks
// into the user DB. Existing rows (matched by youtube_id / station id) are
// left untouched so user data is never overwritten.
func (db *DB) MergeCatalog(catalogPath string) error {
	// read catalog version before attaching
	catConn, err := Open(catalogPath)
	if err != nil {
		return fmt.Errorf("open catalog: %w", err)
	}
	catVersion := catConn.CatalogVersion()
	catConn.Close()

	if catVersion == "" {
		return fmt.Errorf("catalog has no version — may be corrupt")
	}

	_, err = db.conn.Exec(fmt.Sprintf(`ATTACH DATABASE '%s' AS cat`, catalogPath))
	if err != nil {
		return fmt.Errorf("attach catalog: %w", err)
	}
	defer db.conn.Exec("DETACH DATABASE cat")

	_, err = db.conn.Exec(`
		-- ensure catalog tables exist in user DB
		CREATE TABLE IF NOT EXISTS stations (
			id                  TEXT PRIMARY KEY,
			name                TEXT NOT NULL,
			description         TEXT,
			color               TEXT,
			youtube_channel_id  TEXT NOT NULL,
			uploads_playlist_id TEXT NOT NULL,
			last_synced         INTEGER,
			created_at          INTEGER
		);
		CREATE TABLE IF NOT EXISTS tracks (
			id          TEXT PRIMARY KEY,
			station_id  TEXT NOT NULL REFERENCES stations(id),
			youtube_id  TEXT NOT NULL UNIQUE,
			song_title  TEXT,
			artist      TEXT,
			raw_title   TEXT NOT NULL,
			search_text TEXT,
			thumbnail   TEXT,
			published_at INTEGER,
			created_at  INTEGER
		);
		CREATE INDEX IF NOT EXISTS tracks_station_idx   ON tracks(station_id);
		CREATE INDEX IF NOT EXISTS tracks_published_idx ON tracks(published_at DESC);
		CREATE INDEX IF NOT EXISTS tracks_youtube_id_idx ON tracks(youtube_id);
		CREATE VIRTUAL TABLE IF NOT EXISTS tracks_fts USING fts5(
			song_title, artist, raw_title,
			content='tracks', content_rowid='rowid'
		);
	`)
	if err != nil {
		return fmt.Errorf("ensure tables: %w", err)
	}

	_, err = db.conn.Exec(`
		INSERT OR IGNORE INTO stations SELECT * FROM cat.stations;
		INSERT OR IGNORE INTO tracks   SELECT * FROM cat.tracks;
	`)
	if err != nil {
		return fmt.Errorf("merge data: %w", err)
	}

	// rebuild FTS over merged tracks
	db.conn.Exec("INSERT INTO tracks_fts(tracks_fts) VALUES('rebuild')")

	// record the version we just merged
	_, err = db.conn.Exec(
		"INSERT OR REPLACE INTO meta (key, value) VALUES (?, ?)",
		catalogVersionKey, catVersion,
	)
	return err
}

// DefaultCatalogPath returns the system catalog path, falling back to a local
// one beside the binary for development.
func DefaultCatalogPath() string {
	// system install (set by package manager)
	system := "/usr/share/tuitube/catalog.db"
	if _, err := os.Stat(system); err == nil {
		return system
	}
	// dev fallback: catalog.db beside the running binary
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "catalog.db")
}
