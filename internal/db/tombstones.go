package db

import "strings"

// Reasons a track was deliberately removed.
const (
	PruneReasonDead      = "dead"
	PruneReasonDuplicate = "duplicate"
)

// Tombstone records a track that was pruned on purpose.
type Tombstone struct {
	YoutubeID  string
	Reason     string
	ReplacedBy string
	PrunedAt   int64
}

// AddTombstones records tracks as deliberately pruned so sync and the catalog
// merge stop re-adding them. replacedBy is optional and maps a dropped
// youtube_id to the copy that superseded it; it is only meaningful for
// duplicates.
func (db *DB) AddTombstones(youtubeIDs []string, reason string, replacedBy map[string]string) error {
	if len(youtubeIDs) == 0 {
		return nil
	}
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO pruned_tracks (youtube_id, reason, replaced_by)
		VALUES (?, ?, ?)
		ON CONFLICT(youtube_id) DO UPDATE SET
			reason      = excluded.reason,
			replaced_by = COALESCE(excluded.replaced_by, pruned_tracks.replaced_by)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, id := range youtubeIDs {
		var rb any
		if replacedBy != nil {
			if v, ok := replacedBy[id]; ok && v != "" {
				rb = v
			}
		}
		if _, err := stmt.Exec(id, reason, rb); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// TombstonedIDs returns every pruned youtube_id as a set, for cheap lookups
// while walking a channel listing.
func (db *DB) TombstonedIDs() (map[string]struct{}, error) {
	rows, err := db.conn.Query("SELECT youtube_id FROM pruned_tracks")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]struct{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = struct{}{}
	}
	return out, rows.Err()
}

// ListTombstones returns pruned tracks, newest first. reason filters when
// non-empty; limit of 0 means no limit.
func (db *DB) ListTombstones(reason string, limit int) ([]Tombstone, error) {
	q := "SELECT youtube_id, reason, COALESCE(replaced_by,''), pruned_at FROM pruned_tracks"
	var args []any
	if reason != "" {
		q += " WHERE reason = ?"
		args = append(args, reason)
	}
	q += " ORDER BY pruned_at DESC"
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := db.conn.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tombstone
	for rows.Next() {
		var t Tombstone
		if err := rows.Scan(&t.YoutubeID, &t.Reason, &t.ReplacedBy, &t.PrunedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// CountTombstones returns how many tracks are tombstoned, by reason.
func (db *DB) CountTombstones() (map[string]int, error) {
	rows, err := db.conn.Query("SELECT reason, COUNT(*) FROM pruned_tracks GROUP BY reason")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var r string
		var n int
		if err := rows.Scan(&r, &n); err != nil {
			return nil, err
		}
		out[r] = n
	}
	return out, rows.Err()
}

// ForgetTombstones drops tombstones so the next sync may re-add those tracks.
// Passing no ids forgets every tombstone.
func (db *DB) ForgetTombstones(youtubeIDs []string) (int, error) {
	if len(youtubeIDs) == 0 {
		res, err := db.conn.Exec("DELETE FROM pruned_tracks")
		if err != nil {
			return 0, err
		}
		n, err := res.RowsAffected()
		return int(n), err
	}
	args := make([]any, len(youtubeIDs))
	for i, id := range youtubeIDs {
		args[i] = id
	}
	list := "(" + strings.TrimSuffix(strings.Repeat("?,", len(youtubeIDs)), ",") + ")"
	res, err := db.conn.Exec("DELETE FROM pruned_tracks WHERE youtube_id IN "+list, args...)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}
