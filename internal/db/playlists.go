package db

type Playlist struct {
	ID   int64
	Name string
}

func (db *DB) ListPlaylists() ([]Playlist, error) {
	rows, err := db.conn.Query("SELECT id, name FROM playlists ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Playlist
	for rows.Next() {
		var p Playlist
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (db *DB) CreatePlaylist(name string) (int64, error) {
	res, err := db.conn.Exec(
		"INSERT INTO playlists (name) VALUES (?)", name,
	)
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
