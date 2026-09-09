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
	RawTitle   string
	IsFavorite bool
}

// ListTracks queries the library with optional search, favorites filter, limit, and station filter.
// limit=0 means no limit (used by TUI). stationID is optional.
func (db *DB) ListTracks(query string, favoritesOnly bool, limit int, stationID ...string) ([]Track, error) {
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
	limitClause := ""
	if limit > 0 {
		limitClause = " LIMIT ?"
	}

	// build args slice incrementally
	buildArgs := func(base []any) []any {
		if sid != "" {
			base = append(base, sid)
		}
		if limit > 0 {
			base = append(base, limit)
		}
		return base
	}

	switch {
	case query != "":
		ftsQuery := `"` + strings.ReplaceAll(query, `"`, `""`) + `"*`
		favJoin := ""
		queryArgs := []any{favPlaylistID, ftsQuery}
		if favoritesOnly {
			favJoin = " JOIN playlist_tracks fav ON fav.track_id = t.id AND fav.playlist_id = ?"
			queryArgs = []any{favPlaylistID, favPlaylistID, ftsQuery}
		}
		rows, err = db.conn.Query(`
			SELECT t.id, t.youtube_id,
			       COALESCE(NULLIF(t.song_title,''), t.raw_title),
			       COALESCE(t.artist,''),
			       CASE WHEN pt.track_id IS NOT NULL THEN 1 ELSE 0 END
			FROM tracks_fts fts
			JOIN tracks t ON t.rowid = fts.rowid
			LEFT JOIN playlist_tracks pt ON pt.track_id = t.id AND pt.playlist_id = ?
			`+favJoin+`
			WHERE tracks_fts MATCH ?`+stationFilter+`
			ORDER BY rank`+limitClause,
			buildArgs(queryArgs)...)
	case favoritesOnly:
		rows, err = db.conn.Query(`
			SELECT t.id, t.youtube_id,
			       COALESCE(NULLIF(t.song_title,''), t.raw_title),
			       COALESCE(t.artist,''),
			       1
			FROM playlist_tracks pt
			JOIN tracks t ON t.id = pt.track_id
			WHERE pt.playlist_id = ?`+stationFilter+limitClause,
			buildArgs([]any{favPlaylistID})...)
	default:
		rows, err = db.conn.Query(`
			SELECT t.id, t.youtube_id,
			       COALESCE(NULLIF(t.song_title,''), t.raw_title),
			       COALESCE(t.artist,''),
			       CASE WHEN pt.track_id IS NOT NULL THEN 1 ELSE 0 END
			FROM tracks t
			LEFT JOIN playlist_tracks pt ON pt.track_id = t.id AND pt.playlist_id = ?
			WHERE 1=1`+stationFilter+limitClause,
			buildArgs([]any{favPlaylistID})...)
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
