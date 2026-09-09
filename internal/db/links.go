package db

import "strings"

// DeleteTracksByYoutubeID removes tracks and any playlist entries pointing at
// them, returning the number of tracks deleted.
//
// playlist_tracks.track_id has no foreign key to tracks — only playlist_id
// cascades — so those rows are cleared explicitly here; otherwise a prune
// leaves entries referencing tracks that no longer exist.
//
// Callers must RebuildFTS afterwards. tracks_fts is an external-content FTS5
// table with no triggers, so it does not notice deletes and would keep
// returning pruned tracks from search.
func (db *DB) DeleteTracksByYoutubeID(youtubeIDs []string) (int, error) {
	if len(youtubeIDs) == 0 {
		return 0, nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	deleted := 0
	// Chunked to stay clear of SQLite's parameter limit on large prunes.
	const chunk = 400
	for start := 0; start < len(youtubeIDs); start += chunk {
		end := min(start+chunk, len(youtubeIDs))
		batch := youtubeIDs[start:end]

		args := make([]any, len(batch))
		for i, id := range batch {
			args[i] = id
		}
		list := "(" + strings.TrimSuffix(strings.Repeat("?,", len(batch)), ",") + ")"

		if _, err := tx.Exec(
			`DELETE FROM playlist_tracks
			  WHERE track_id IN (SELECT id FROM tracks WHERE youtube_id IN `+list+`)`,
			args...,
		); err != nil {
			return 0, err
		}

		res, err := tx.Exec(`DELETE FROM tracks WHERE youtube_id IN `+list, args...)
		if err != nil {
			return 0, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return 0, err
		}
		deleted += int(n)
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return deleted, nil
}

// DedupeCandidate carries the fields needed to judge duplicate tracks.
type DedupeCandidate struct {
	YoutubeID   string
	Artist      string
	SongTitle   string
	RawTitle    string
	PublishedAt int64
	Station     string
}

// DedupeCandidates returns every track with the station name attached,
// optionally limited to one station.
func (db *DB) DedupeCandidates(stationID string) ([]DedupeCandidate, error) {
	query := `
		SELECT t.youtube_id,
		       COALESCE(t.artist,''),
		       COALESCE(NULLIF(t.song_title,''), t.raw_title),
		       t.raw_title,
		       COALESCE(t.published_at,0),
		       COALESCE(s.name,'')
		FROM tracks t
		JOIN stations s ON s.id = t.station_id`
	var args []any
	if stationID != "" {
		query += " WHERE t.station_id = ?"
		args = append(args, stationID)
	}
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DedupeCandidate
	for rows.Next() {
		var c DedupeCandidate
		if err := rows.Scan(&c.YoutubeID, &c.Artist, &c.SongTitle, &c.RawTitle, &c.PublishedAt, &c.Station); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// RemapPlaylistTracks repoints playlist entries from duplicate tracks about to
// be deleted onto the copy being kept, returning the number of entries moved.
//
// Without this a dedupe silently shortens hand-curated playlists: the entry
// points at a dropped copy, so deleting it takes the song out of the playlist
// even though the same recording survives under another youtube_id.
//
// remap maps the youtube_id being dropped to the one being kept. INSERT OR
// IGNORE covers a playlist that already contains the keeper, and the original
// added_at is preserved so playlist ordering is unchanged.
func (db *DB) RemapPlaylistTracks(remap map[string]string) (int, error) {
	if len(remap) == 0 {
		return 0, nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO playlist_tracks (playlist_id, track_id, added_at)
		SELECT pt.playlist_id, keep.id, pt.added_at
		FROM playlist_tracks pt
		JOIN tracks dropped ON dropped.id = pt.track_id AND dropped.youtube_id = ?
		JOIN tracks keep    ON keep.youtube_id = ?`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	moved := 0
	for dropID, keepID := range remap {
		res, err := stmt.Exec(dropID, keepID)
		if err != nil {
			return 0, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return 0, err
		}
		moved += int(n)
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return moved, nil
}
