// Package enrich fetches music metadata from Last.fm and stores it in the DB.
package enrich

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gitcoder89431/tui-tube/internal/db"
)

const (
	lastfmBase   = "https://ws.audioscrobbler.com/2.0/"
	concurrency  = 5   // parallel requests
	minTagCount  = 5   // ignore tags with fewer votes
	maxTags      = 6   // tags to store per track
	batchPause   = 250 * time.Millisecond // pause between batches
)

type lastfmTag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type lastfmResponse struct {
	TopTags struct {
		Tag []lastfmTag `json:"tag"`
	} `json:"toptags"`
	Error   int    `json:"error"`
	Message string `json:"message"`
}

// Run enriches all unenriched tracks in the DB with Last.fm tags.
// Progress is printed to stdout. Resumable — already-tagged tracks are skipped.
func Run(database *db.DB, apiKey string, w io.Writer) error {
	if err := database.Migrate(); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	tracks, err := database.UnenrichedTracks()
	if err != nil {
		return fmt.Errorf("list tracks: %w", err)
	}

	total := len(tracks)
	if total == 0 {
		fmt.Fprintln(w, "all tracks already enriched")
		return nil
	}
	fmt.Fprintf(w, "enriching %d tracks (concurrency=%d)...\n", total, concurrency)

	var done atomic.Int64
	var mu sync.Mutex
	sem := make(chan struct{}, concurrency)
	errs := make(chan error, total)

	for _, t := range tracks {
		t := t
		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()

			tags, err := fetchTags(apiKey, t.Artist, t.SongTitle)
			if err != nil {
				// non-fatal: store empty tags so we don't retry forever
				tags = []string{}
			}

			mu.Lock()
			_ = database.SetTags(t.ID, tags)
			mu.Unlock()

			n := done.Add(1)
			if n%100 == 0 || n == int64(total) {
				fmt.Fprintf(w, "  %d/%d\n", n, total)
			}
			time.Sleep(batchPause / concurrency)
		}()
	}

	// drain semaphore
	for i := 0; i < concurrency; i++ {
		sem <- struct{}{}
	}
	close(errs)

	return nil
}

func fetchTags(apiKey, artist, title string) ([]string, error) {
	params := url.Values{
		"method":      {"track.getTopTags"},
		"artist":      {artist},
		"track":       {title},
		"api_key":     {apiKey},
		"format":      {"json"},
		"autocorrect": {"1"},
	}
	resp, err := http.Get(lastfmBase + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result lastfmResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != 0 {
		return nil, fmt.Errorf("lastfm error %d: %s", result.Error, result.Message)
	}

	var tags []string
	for _, tag := range result.TopTags.Tag {
		if tag.Count < minTagCount {
			continue
		}
		tags = append(tags, tag.Name)
		if len(tags) >= maxTags {
			break
		}
	}
	return tags, nil
}
