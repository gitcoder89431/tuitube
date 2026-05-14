package db

import (
	"database/sql"
	"strings"
)

const favPlaylistID = 1

type Track struct {
	ID         string
	YoutubeID  string
	SongTitle  string
	Artist     string
	IsFavorite bool
}

func (db *DB) ListTracks(query string, favoritesOnly bool, stationID ...string) ([]Track, error) {
	var rows *sql.Rows
	var err error

	sid := ""
	if len(stationID) > 0 {
		sid = stationID[0]
	}

	stationFilter := ""
	if sid != "" {
		stationFilter = " AND t.station_id = ?"
	}
	stationArg := func(args ...any) []any {
		if sid != "" {
			return append(args, sid)
		}
		return args
	}

	switch {
	case query != "":
		ftsQuery := `"` + strings.ReplaceAll(query, `"`, `""`) + `"*`
		rows, err = db.conn.Query(`
			SELECT t.id, t.youtube_id,
			       COALESCE(NULLIF(t.song_title,''), t.raw_title),
			       COALESCE(t.artist,''),
			       CASE WHEN pt.track_id IS NOT NULL THEN 1 ELSE 0 END
			FROM tracks_fts fts
			JOIN tracks t ON t.rowid = fts.rowid
			LEFT JOIN playlist_tracks pt ON pt.track_id = t.id AND pt.playlist_id = ?
			WHERE tracks_fts MATCH ?`+stationFilter+`
			ORDER BY rank
		`, stationArg(favPlaylistID, ftsQuery)...)
	case favoritesOnly:
		rows, err = db.conn.Query(`
			SELECT t.id, t.youtube_id,
			       COALESCE(NULLIF(t.song_title,''), t.raw_title),
			       COALESCE(t.artist,''),
			       1
			FROM playlist_tracks pt
			JOIN tracks t ON t.id = pt.track_id
			WHERE pt.playlist_id = ?`+stationFilter+`
			ORDER BY pt.added_at DESC
		`, stationArg(favPlaylistID)...)
	default:
		rows, err = db.conn.Query(`
			SELECT t.id, t.youtube_id,
			       COALESCE(NULLIF(t.song_title,''), t.raw_title),
			       COALESCE(t.artist,''),
			       CASE WHEN pt.track_id IS NOT NULL THEN 1 ELSE 0 END
			FROM tracks t
			LEFT JOIN playlist_tracks pt ON pt.track_id = t.id AND pt.playlist_id = ?
			WHERE 1=1`+stationFilter+`
			ORDER BY t.published_at DESC
		`, stationArg(favPlaylistID)...)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []Track
	for rows.Next() {
		var t Track
		var isFav int
		if err := rows.Scan(&t.ID, &t.YoutubeID, &t.SongTitle, &t.Artist, &isFav); err != nil {
			return nil, err
		}
		t.IsFavorite = isFav == 1
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
