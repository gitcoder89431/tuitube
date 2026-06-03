package db

import (
	"path/filepath"
	"testing"
)

func openMemDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.InitUserDB(); err != nil {
		t.Fatalf("init db: %v", err)
	}
	return db
}

func seedStation(t *testing.T, db *DB, id, name string) {
	t.Helper()
	_, err := db.conn.Exec(
		`INSERT INTO stations (id, name, youtube_channel_id, uploads_playlist_id) VALUES (?, ?, 'ch1', 'pl1')`,
		id, name,
	)
	if err != nil {
		t.Fatalf("seed station: %v", err)
	}
}

func seedTrack(t *testing.T, db *DB, id, youtubeID, stationID string) {
	t.Helper()
	_, err := db.conn.Exec(
		`INSERT INTO tracks (id, station_id, youtube_id, raw_title) VALUES (?, ?, ?, 'raw')`,
		id, stationID, youtubeID,
	)
	if err != nil {
		t.Fatalf("seed track: %v", err)
	}
}

func TestToggleFavoriteRoundTrip(t *testing.T) {
	db := openMemDB(t)
	seedStation(t, db, "s1", "Station 1")
	seedTrack(t, db, "t1", "yt_abc", "s1")

	// toggle on
	isFav, err := db.ToggleFavorite("t1")
	if err != nil {
		t.Fatalf("toggle on: %v", err)
	}
	if !isFav {
		t.Error("expected isFav=true after first toggle")
	}

	var count int
	db.conn.QueryRow("SELECT COUNT(*) FROM playlist_tracks WHERE track_id=? AND playlist_id=1", "t1").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 row in playlist_tracks, got %d", count)
	}

	// toggle off
	isFav, err = db.ToggleFavorite("t1")
	if err != nil {
		t.Fatalf("toggle off: %v", err)
	}
	if isFav {
		t.Error("expected isFav=false after second toggle")
	}

	db.conn.QueryRow("SELECT COUNT(*) FROM playlist_tracks WHERE track_id=? AND playlist_id=1", "t1").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 rows in playlist_tracks after toggle off, got %d", count)
	}
}

func TestValidateCatalogPath(t *testing.T) {
	cases := []struct {
		path    string
		wantErr bool
	}{
		{"catalog.db", false},
		{"/some/path/catalog.db", false},
		{"catalog.txt", true},
		{"cat'alog.db", true},
		{"cat;alog.db", true},
		{"", true},
	}
	for _, c := range cases {
		err := validateCatalogPath(c.path)
		if (err != nil) != c.wantErr {
			t.Errorf("validateCatalogPath(%q): got err=%v, wantErr=%v", c.path, err, c.wantErr)
		}
	}
}
