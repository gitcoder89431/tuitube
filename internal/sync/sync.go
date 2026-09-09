package sync

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/gitcoder89431/tuitube/internal/db"
)

type ytVideo struct {
	ID                string  `json:"id"`
	Title             string  `json:"title"`
	Thumbnail         string  `json:"thumbnail"`
	Timestamp         float64 `json:"timestamp"`
	ChannelID         string  `json:"channel_id"`
	UploaderID        string  `json:"uploader_id"`
	PlaylistID        string  `json:"playlist_id"`
	PlaylistChannelID string  `json:"playlist_channel_id"`
}

// Station syncs a single station by its uploads playlist URL.
// Prints progress to w. Returns count of newly inserted tracks.
func Station(database *db.DB, station db.Station, w io.Writer) (int, error) {
	url := "https://www.youtube.com/playlist?list=" + station.UploadsPlaylistID

	fmt.Fprintf(w, "  fetching %s...\n", station.Name)
	cmd := exec.Command("yt-dlp", "--flat-playlist", "--dump-json", url)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("yt-dlp: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	fmt.Fprintf(w, "  found %d videos\n", len(lines))

	inserted := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var v ytVideo
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			continue
		}

		exists, err := database.TrackExistsByYoutubeID(v.ID)
		if err != nil {
			return inserted, err
		}
		if exists {
			continue
		}

		artist, songTitle := parseTitle(v.Title)
		artist = CleanArtist(artist)
		thumbnail := v.Thumbnail
		var publishedAt int64
		if v.Timestamp > 0 {
			publishedAt = int64(v.Timestamp) * 1000
		}

		ok, err := database.InsertTrack(station.ID, v.ID, v.Title, artist, songTitle, thumbnail, publishedAt)
		if err != nil {
			return inserted, fmt.Errorf("insert %s: %w", v.ID, err)
		}
		if ok {
			inserted++
			if inserted%50 == 0 {
				fmt.Fprintf(w, "  %d inserted...\n", inserted)
			}
		}
	}

	if err := database.UpdateStationSyncTime(station.ID); err != nil {
		return inserted, err
	}

	return inserted, nil
}

// DiscoverStation fetches channel metadata from a channel URL and returns
// a Station ready to be upserted. The station ID is derived from the channel ID.
func DiscoverStation(channelURL string, w io.Writer) (db.Station, error) {
	fmt.Fprintf(w, "  probing %s...\n", channelURL)
	cmd := exec.Command("yt-dlp",
		"--flat-playlist", "--dump-json",
		"--playlist-items", "1",
		channelURL,
	)
	out, err := cmd.Output()
	if err != nil {
		return db.Station{}, fmt.Errorf("yt-dlp probe: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return db.Station{}, fmt.Errorf("no output from yt-dlp")
	}

	var v ytVideo
	if err := json.Unmarshal([]byte(lines[0]), &v); err != nil {
		return db.Station{}, fmt.Errorf("parse probe: %w", err)
	}

	// channel_id comes from full extraction; flat-playlist puts it in playlist_channel_id
	channelID := v.ChannelID
	if channelID == "" {
		channelID = v.PlaylistChannelID
	}
	if channelID == "" {
		channelID = v.UploaderID
	}
	if channelID == "" {
		return db.Station{}, fmt.Errorf("could not determine channel_id from yt-dlp output")
	}
	// If we got a handle (@name) instead of a UC-id, do a second probe via the channel URL
	if strings.HasPrefix(channelID, "@") {
		fmt.Fprintf(w, "  got handle %s, re-probing for channel ID...\n", channelID)
		cmd2 := exec.Command("yt-dlp",
			"--flat-playlist", "--dump-json",
			"--playlist-items", "1",
			"https://www.youtube.com/"+channelID+"/videos",
		)
		out2, err := cmd2.Output()
		if err != nil {
			return db.Station{}, fmt.Errorf("yt-dlp re-probe for handle: %w", err)
		}
		lines2 := strings.Split(strings.TrimSpace(string(out2)), "\n")
		if len(lines2) > 0 {
			var v2 ytVideo
			if err := json.Unmarshal([]byte(lines2[0]), &v2); err == nil {
				if v2.ChannelID != "" {
					channelID = v2.ChannelID
				} else if v2.PlaylistChannelID != "" {
					channelID = v2.PlaylistChannelID
				}
			}
		}
	}
	// Normalise: all YouTube channel IDs start with UC; uploads playlist is UU + rest
	if len(channelID) < 2 || channelID[:2] != "UC" {
		return db.Station{}, fmt.Errorf("unexpected channel_id format: %q", channelID)
	}
	uploadsPlaylistID := "UU" + channelID[2:]

	return db.Station{
		ID:                channelID,
		YoutubeChannelID:  channelID,
		UploadsPlaylistID: uploadsPlaylistID,
	}, nil
}

// parseTitle cleans the raw title then splits it into artist and song.
func parseTitle(raw string) (artist, songTitle string) {
	return SplitArtistTitle(CleanTitle(raw))
}
