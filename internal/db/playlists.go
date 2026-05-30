package db

type Playlist struct {
	ID         int64
	Name       string
	TrackCount int
}

type StationSummary struct {
	ID         string
	Name       string
	TrackCount int
}

func (db *DB) ListPlaylists() ([]Playlist, error) {
	rows, err := db.conn.Query(`
		SELECT p.id, p.name, COUNT(pt.track_id)
		FROM playlists p
		LEFT JOIN playlist_tracks pt ON pt.playlist_id = p.id
		GROUP BY p.id ORDER BY p.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Playlist
	for rows.Next() {
		var p Playlist
		if err := rows.Scan(&p.ID, &p.Name, &p.TrackCount); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (db *DB) ListStationSummaries() ([]StationSummary, error) {
	rows, err := db.conn.Query(`
		SELECT s.id, s.name, COUNT(t.id)
		FROM stations s
		LEFT JOIN tracks t ON t.station_id = s.id
		GROUP BY s.id ORDER BY s.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StationSummary
	for rows.Next() {
		var s StationSummary
		if err := rows.Scan(&s.ID, &s.Name, &s.TrackCount); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (db *DB) CreatePlaylist(name string) (int64, error) {
	res, err := db.conn.Exec("INSERT INTO playlists (name) VALUES (?)", name)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) AddToPlaylist(playlistID int64, trackID string) error {
	_, err := db.conn.Exec(`
		INSERT OR IGNORE INTO playlist_tracks (playlist_id, track_id)
		VALUES (?, ?)
	`, playlistID, trackID)
	return err
}

// AddToPlaylistBatch inserts multiple tracks in a single transaction.
// All-or-nothing: rolls back if any insert fails.
func (db *DB) AddToPlaylistBatch(playlistID int64, trackIDs []string) (int, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO playlist_tracks (playlist_id, track_id) VALUES (?, ?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	added := 0
	for _, id := range trackIDs {
		res, err := stmt.Exec(playlistID, id)
		if err != nil {
			return 0, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			added++
		}
	}
	return added, tx.Commit()
}

func (db *DB) RemoveFromPlaylist(playlistID int64, trackID string) error {
	_, err := db.conn.Exec(
		"DELETE FROM playlist_tracks WHERE playlist_id=? AND track_id=?",
		playlistID, trackID,
	)
	return err
}

func (db *DB) ListPlaylistTracks(playlistID int64) ([]Track, error) {
	rows, err := db.conn.Query(`
		SELECT t.id, t.youtube_id,
		       COALESCE(NULLIF(t.song_title,''), t.raw_title),
		       COALESCE(t.artist,''),
		       1
		FROM playlist_tracks pt
		JOIN tracks t ON t.id = pt.track_id
		WHERE pt.playlist_id = ?
		ORDER BY pt.added_at DESC
	`, playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTracks(rows)
}

func (db *DB) ListStationTracks(stationID string) ([]Track, error) {
	rows, err := db.conn.Query(`
		SELECT t.id, t.youtube_id,
		       COALESCE(NULLIF(t.song_title,''), t.raw_title),
		       COALESCE(t.artist,''),
		       CASE WHEN pt.track_id IS NOT NULL THEN 1 ELSE 0 END
		FROM tracks t
		LEFT JOIN playlist_tracks pt ON pt.track_id = t.id AND pt.playlist_id = 1
		WHERE t.station_id = ?
		ORDER BY t.published_at DESC
	`, stationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTracks(rows)
}

func scanTracks(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close() error
}) ([]Track, error) {
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
