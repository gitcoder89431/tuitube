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
