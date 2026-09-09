package db

import "testing"

// InsertTrack builds its VALUES list by hand, so a column change can leave the
// placeholder count out of step with the column list. That mismatch only
// surfaces on a real insert — not on schema setup or a catalog merge — so it
// needs direct coverage.
func TestInsertTrackRoundTrip(t *testing.T) {
	d := openMemDB(t)
	seedStation(t, d, "st1", "Station One")

	ok, err := d.InsertTrack("st1", "yt123", "Artist - Song", "Artist", "Song",
		"https://example.invalid/t.jpg", 1700000000000)
	if err != nil {
		t.Fatalf("InsertTrack: %v", err)
	}
	if !ok {
		t.Fatal("InsertTrack reported no row inserted")
	}

	var artist, song, raw, search, thumb string
	var published int64
	err = d.QueryRow(`SELECT artist, song_title, raw_title, search_text,
	                         thumbnail, published_at
	                  FROM tracks WHERE youtube_id = ?`, "yt123").
		Scan(&artist, &song, &raw, &search, &thumb, &published)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	// Each value must land in its own column, not shifted by one.
	if artist != "Artist" || song != "Song" || raw != "Artist - Song" {
		t.Errorf("columns misaligned: artist=%q song=%q raw=%q", artist, song, raw)
	}
	if search != "artist song" {
		t.Errorf("search_text = %q, want %q", search, "artist song")
	}
	if thumb != "https://example.invalid/t.jpg" {
		t.Errorf("thumbnail = %q", thumb)
	}
	if published != 1700000000000 {
		t.Errorf("published_at = %d, want 1700000000000", published)
	}
}

// The youtube_id UNIQUE constraint makes a re-insert a no-op, not an error.
func TestInsertTrackDuplicateIsNoOp(t *testing.T) {
	d := openMemDB(t)
	seedStation(t, d, "st1", "Station One")
	if _, err := d.InsertTrack("st1", "yt123", "raw", "A", "S", "", 0); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	ok, err := d.InsertTrack("st1", "yt123", "raw", "A", "S", "", 0)
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	if ok {
		t.Error("second insert reported a row inserted, want no-op")
	}
}
