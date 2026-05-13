package db

import (
	"database/sql"
	"encoding/json"
)

type Station struct {
	ID               string
	Name             string
	YoutubeChannelID string
	UploadsPlaylistID string
}

func (db *DB) ListStations() ([]Station, error) {
	rows, err := db.conn.Query(`
		SELECT id, name, youtube_channel_id, uploads_playlist_id
		FROM stations ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var stations []Station
	for rows.Next() {
		var s Station
		if err := rows.Scan(&s.ID, &s.Name, &s.YoutubeChannelID, &s.UploadsPlaylistID); err != nil {
			return nil, err
		}
		stations = append(stations, s)
	}
	return stations, rows.Err()
}

func (db *DB) UpsertStation(s Station) error {
	_, err := db.conn.Exec(`
		INSERT INTO stations (id, name, youtube_channel_id, uploads_playlist_id, created_at)
		VALUES (?, ?, ?, ?, strftime('%s','now')*1000)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,
			youtube_channel_id=excluded.youtube_channel_id,
			uploads_playlist_id=excluded.uploads_playlist_id
	`, s.ID, s.Name, s.YoutubeChannelID, s.UploadsPlaylistID)
	return err
}

func (db *DB) InsertTrack(stationID, youtubeID, rawTitle, artist, songTitle, thumbnail string, publishedAt int64) (inserted bool, err error) {
	res, err := db.conn.Exec(`
		INSERT OR IGNORE INTO tracks
			(id, station_id, youtube_id, raw_title, artist, song_title, search_text, thumbnail, published_at, created_at)
		VALUES (
			lower(hex(randomblob(16))),
			?, ?, ?, ?, ?,
			lower(? || ' ' || ?),
			?,
			?,
			strftime('%s','now')*1000
		)
	`, stationID, youtubeID, rawTitle, artist, songTitle, artist, songTitle, thumbnail, publishedAt)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (db *DB) RebuildFTS() error {
	_, err := db.conn.Exec("INSERT INTO tracks_fts(tracks_fts) VALUES('rebuild')")
	return err
}

func (db *DB) TrackExistsByYoutubeID(youtubeID string) (bool, error) {
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM tracks WHERE youtube_id=?", youtubeID).Scan(&count)
	return count > 0, err
}

func (db *DB) UpdateStationSyncTime(stationID string) error {
	_, err := db.conn.Exec(
		"UPDATE stations SET last_synced=strftime('%s','now')*1000 WHERE id=?",
		stationID,
	)
	return err
}

// UnenrichedTracks returns all tracks that have no tags yet.
func (db *DB) UnenrichedTracks() ([]Track, error) {
	rows, err := db.conn.Query(`
		SELECT id, youtube_id,
		       COALESCE(NULLIF(song_title,''), raw_title),
		       COALESCE(artist,''), 0
		FROM tracks WHERE tags IS NULL OR tags = ''
	`)
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
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

// SetTags stores a JSON array of tags for a track.
func (db *DB) SetTags(trackID string, tags []string) error {
	b, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	_, err = db.conn.Exec("UPDATE tracks SET tags=? WHERE id=?", string(b), trackID)
	return err
}

// UpdateTrackTitles updates the song_title and artist for a single track.
func (db *DB) UpdateTrackTitles(id, songTitle, artist string) error {
	_, err := db.conn.Exec(
		"UPDATE tracks SET song_title=?, artist=?, search_text=lower(?||' '||?) WHERE id=?",
		songTitle, artist, artist, songTitle, id,
	)
	return err
}

// AllTracks returns every track id + raw fields needed for cleanup.
func (db *DB) AllTracks() ([]Track, error) {
	rows, err := db.conn.Query(`
		SELECT id, youtube_id,
		       COALESCE(NULLIF(song_title,''), raw_title),
		       COALESCE(artist,''),
		       0
		FROM tracks
	`)
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
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

// StationByPlaylistID looks up a station by its uploads_playlist_id.
// Returns sql.ErrNoRows if not found.
func (db *DB) StationByPlaylistID(playlistID string) (Station, error) {
	var s Station
	err := db.conn.QueryRow(`
		SELECT id, name, youtube_channel_id, uploads_playlist_id
		FROM stations WHERE uploads_playlist_id=?
	`, playlistID).Scan(&s.ID, &s.Name, &s.YoutubeChannelID, &s.UploadsPlaylistID)
	if err == sql.ErrNoRows {
		return Station{}, sql.ErrNoRows
	}
	return s, err
}
