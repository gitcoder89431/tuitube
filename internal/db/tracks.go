package db

import (
	"database/sql"
	"encoding/json"
	"strings"
)

const favPlaylistID = 1

type Track struct {
	ID         string
	YoutubeID  string
	SongTitle  string
	Artist     string
	IsFavorite bool
	Tags       []string
}

func (db *DB) ListTracks(query string, favoritesOnly bool) ([]Track, error) {
	var rows *sql.Rows
	var err error

	switch {
	case query != "":
		ftsQuery := `"` + strings.ReplaceAll(query, `"`, `""`) + `"*`
		rows, err = db.conn.Query(`
			SELECT t.id, t.youtube_id,
			       COALESCE(NULLIF(t.song_title,''), t.raw_title),
			       COALESCE(t.artist,''),
			       CASE WHEN pt.track_id IS NOT NULL THEN 1 ELSE 0 END,
			       COALESCE(t.tags,'')
			FROM tracks_fts fts
			JOIN tracks t ON t.rowid = fts.rowid
			LEFT JOIN playlist_tracks pt ON pt.track_id = t.id AND pt.playlist_id = ?
			WHERE tracks_fts MATCH ?
			ORDER BY rank
		`, favPlaylistID, ftsQuery)
	case favoritesOnly:
		rows, err = db.conn.Query(`
			SELECT t.id, t.youtube_id,
			       COALESCE(NULLIF(t.song_title,''), t.raw_title),
			       COALESCE(t.artist,''),
			       1,
			       COALESCE(t.tags,'')
			FROM playlist_tracks pt
			JOIN tracks t ON t.id = pt.track_id
			WHERE pt.playlist_id = ?
			ORDER BY pt.added_at DESC
		`, favPlaylistID)
	default:
		rows, err = db.conn.Query(`
			SELECT t.id, t.youtube_id,
			       COALESCE(NULLIF(t.song_title,''), t.raw_title),
			       COALESCE(t.artist,''),
			       CASE WHEN pt.track_id IS NOT NULL THEN 1 ELSE 0 END,
			       COALESCE(t.tags,'')
			FROM tracks t
			LEFT JOIN playlist_tracks pt ON pt.track_id = t.id AND pt.playlist_id = ?
			ORDER BY t.published_at DESC
		`, favPlaylistID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []Track
	for rows.Next() {
		var t Track
		var isFav int
		var tagsJSON string
		if err := rows.Scan(&t.ID, &t.YoutubeID, &t.SongTitle, &t.Artist, &isFav, &tagsJSON); err != nil {
			return nil, err
		}
		t.IsFavorite = isFav == 1
		if tagsJSON != "" {
			json.Unmarshal([]byte(tagsJSON), &t.Tags)
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func (db *DB) ToggleFavorite(trackID string) (bool, error) {
	var count int
	if err := db.conn.QueryRow(
		"SELECT COUNT(*) FROM playlist_tracks WHERE playlist_id=? AND track_id=?",
		favPlaylistID, trackID,
	).Scan(&count); err != nil {
		return false, err
	}
	if count > 0 {
		_, err := db.conn.Exec(
			"DELETE FROM playlist_tracks WHERE playlist_id=? AND track_id=?",
			favPlaylistID, trackID,
		)
		return false, err
	}
	_, err := db.conn.Exec(
		"INSERT INTO playlist_tracks (playlist_id, track_id, added_at) VALUES (?, ?, strftime('%s','now')*1000)",
		favPlaylistID, trackID,
	)
	return true, err
}
