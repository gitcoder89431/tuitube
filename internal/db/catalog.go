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

// MergeResult summarises what changed during a catalog merge.
type MergeResult struct {
	NewStations int
	NewTracks   int
	NewPlaylists int
	CatalogVersion string
}

// MergeCatalog attaches catalogPath and merges any new stations and tracks
// into the user DB. Existing rows (matched by youtube_id / station id) are
// left untouched so user data is never overwritten.
func (db *DB) MergeCatalog(catalogPath string) error {
	_, err := db.MergeCatalogFull(catalogPath, false)
	return err
}

// MergeCatalogFull merges the catalog and returns a summary of changes.
// Pass dryRun=true to see what would change without writing anything.
func (db *DB) MergeCatalogFull(catalogPath string, dryRun bool) (MergeResult, error) {
	var result MergeResult

	catConn, err := Open(catalogPath)
	if err != nil {
		return result, fmt.Errorf("open catalog: %w", err)
	}
	catVersion := catConn.CatalogVersion()
	catConn.Close()

	if catVersion == "" {
		return result, fmt.Errorf("catalog has no version — may be corrupt")
	}
	result.CatalogVersion = catVersion

	if dryRun {
		// count what would change without writing
		tmpConn, err := Open(catalogPath)
		if err != nil {
			return result, err
		}
		defer tmpConn.Close()
		tmpConn.conn.QueryRow("SELECT COUNT(*) FROM stations").Scan(&result.NewStations)
		tmpConn.conn.QueryRow("SELECT COUNT(*) FROM tracks").Scan(&result.NewTracks)
		return result, nil
	}

	_, err = db.conn.Exec(fmt.Sprintf(`ATTACH DATABASE '%s' AS cat`, catalogPath))
	if err != nil {
		return result, fmt.Errorf("attach catalog: %w", err)
	}
	defer db.conn.Exec("DETACH DATABASE cat")

	if err := db.InitUserDB(); err != nil {
		return result, fmt.Errorf("ensure tables: %w", err)
	}

	// count new stations before insert
	db.conn.QueryRow(`SELECT COUNT(*) FROM cat.stations WHERE id NOT IN (SELECT id FROM stations)`).Scan(&result.NewStations)
	db.conn.QueryRow(`SELECT COUNT(*) FROM cat.tracks WHERE youtube_id NOT IN (SELECT youtube_id FROM tracks)`).Scan(&result.NewTracks)

	_, err = db.conn.Exec(`
		INSERT OR IGNORE INTO stations SELECT * FROM cat.stations;
		INSERT OR IGNORE INTO tracks   SELECT * FROM cat.tracks;
	`)
	if err != nil {
		return result, fmt.Errorf("merge data: %w", err)
	}

	if result.NewTracks > 0 {
		db.conn.Exec("INSERT INTO tracks_fts(tracks_fts) VALUES('rebuild')")
	}

	newPL, _ := db.seedDemoPlaylists()
	result.NewPlaylists = newPL

	_, err = db.conn.Exec(
		"INSERT OR REPLACE INTO meta (key, value) VALUES (?, ?)",
		catalogVersionKey, catVersion,
	)
	return result, err
}

// seedDemoPlaylists reads demo_playlists from the attached catalog and creates
// any that don't already exist in the user's playlists table.
func (db *DB) seedDemoPlaylists() (int, error) {
	// check if catalog has demo tables
	var n int
	err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM cat.sqlite_master WHERE type='table' AND name='demo_playlists'",
	).Scan(&n)
	if err != nil || n == 0 {
		return 0, nil // older catalog, no demo playlists
	}

	rows, err := db.conn.Query("SELECT name FROM cat.demo_playlists")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	created := 0
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return created, err
		}
		var count int
		db.conn.QueryRow("SELECT COUNT(*) FROM playlists WHERE name=?", name).Scan(&count)
		if count > 0 {
			continue
		}
		res, err := db.conn.Exec("INSERT INTO playlists (name) VALUES (?)", name)
		if err != nil {
			return created, err
		}
		pid, _ := res.LastInsertId()
		_, err = db.conn.Exec(`
			INSERT OR IGNORE INTO playlist_tracks (playlist_id, track_id)
			SELECT ?, t.id FROM cat.demo_playlist_tracks dpt
			JOIN tracks t ON t.youtube_id = dpt.youtube_id
			WHERE dpt.playlist_name = ?
		`, pid, name)
		if err != nil {
			return created, err
		}
		created++
	}
	return created, rows.Err()
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
