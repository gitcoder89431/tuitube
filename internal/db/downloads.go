package db

import "os"

type Download struct {
	YoutubeID string
	Filepath  string
}

func (db *DB) InitDownloads() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS downloads (
			youtube_id  TEXT PRIMARY KEY,
			filepath    TEXT NOT NULL,
			downloaded_at INTEGER NOT NULL DEFAULT (strftime('%s','now')*1000)
		)
	`)
	return err
}

func (db *DB) MarkDownloaded(youtubeID, filepath string) error {
	_, err := db.conn.Exec(`
		INSERT OR REPLACE INTO downloads (youtube_id, filepath) VALUES (?, ?)
	`, youtubeID, filepath)
	return err
}

// LoadDownloaded returns a map of youtube ID → local filepath for files that still exist on disk.
func (db *DB) LoadDownloaded() (map[string]string, error) {
	rows, err := db.conn.Query("SELECT youtube_id, filepath FROM downloads")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	var toDelete []string

	for rows.Next() {
		var id, fp string
		if err := rows.Scan(&id, &fp); err != nil {
			return nil, err
		}
		if _, err := os.Stat(fp); err == nil {
			result[id] = fp
		} else {
			toDelete = append(toDelete, id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// clean up stale entries (file moved or deleted)
	for _, id := range toDelete {
		_, _ = db.conn.Exec("DELETE FROM downloads WHERE youtube_id=?", id)
	}

	return result, nil
}
