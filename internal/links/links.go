// Package links checks whether the YouTube videos behind tracks are still
// playable, so dead entries can be pruned before they reach a client.
//
// Checking runs in two stages. The oEmbed endpoint is a cheap HTTP probe that
// clears the overwhelming majority of tracks in one request each, but its
// status codes are not trustworthy on their own: a 403 can mean "embedding
// disabled" for a video that plays perfectly well. So anything the probe does
// not return 200 for is handed to yt-dlp — the same resolver mpv uses — and
// only its verdict decides whether a track is dead.
package links

import (
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Status string

const (
	// StatusAlive means the video resolved and can be played.
	StatusAlive Status = "alive"
	// StatusDead means the video is gone for everyone: deleted, private, or
	// taken down. Safe to prune.
	StatusDead Status = "dead"
	// StatusUnknown means the check could not reach a verdict — a network
	// failure, a throttled request, or a region block that may still play
	// elsewhere. Never pruned.
	StatusUnknown Status = "unknown"
)

type Result struct {
	YoutubeID string `json:"youtube_id"`
	Title     string `json:"title,omitempty"`
	Status    Status `json:"status"`
	Reason    string `json:"reason,omitempty"`
}

// Track is the subset of a db.Track this package needs, kept local so the
// checker does not depend on the db package.
type Track struct {
	YoutubeID string
	Title     string
}

// deadPatterns appear in yt-dlp errors for videos that are gone for everyone.
var deadPatterns = []string{
	"private video",
	"video unavailable",
	"this video is unavailable",
	"video has been removed",
	"removed by the uploader",
	"account associated with this video has been terminated",
	"video is no longer available",
	"this video has been removed",
	"video does not exist",
}

// regionPatterns also surface as "unavailable" but only for us — the video may
// play fine from another country, so these must never be pruned from a library
// that gets served to other people.
var regionPatterns = []string{
	"not available in your country",
	"blocked it in your country",
	"not available from your location",
	"who has blocked it on copyright grounds",
}

// classify turns a yt-dlp error into a verdict. Anything not positively
// recognised as dead stays unknown, so an unfamiliar or transient failure can
// never cause a delete.
func classify(ytdlpErr string) (Status, string) {
	msg := strings.TrimSpace(ytdlpErr)
	// Strip yt-dlp's "ERROR: [youtube] <id>: " prefix for a readable reason.
	msg = strings.TrimPrefix(msg, "ERROR: ")
	if strings.HasPrefix(msg, "[") {
		if i := strings.Index(msg, "] "); i != -1 {
			rest := msg[i+2:]
			// What follows the extractor tag is "<id>: message".
			if j := strings.Index(rest, ": "); j != -1 {
				rest = rest[j+2:]
			}
			msg = rest
		}
	}
	msg = strings.TrimSpace(msg)

	low := strings.ToLower(msg)
	for _, p := range regionPatterns {
		if strings.Contains(low, p) {
			return StatusUnknown, msg
		}
	}
	for _, p := range deadPatterns {
		if strings.Contains(low, p) {
			return StatusDead, msg
		}
	}
	return StatusUnknown, msg
}

type Checker struct {
	// ProbeConcurrency bounds the cheap oEmbed stage.
	ProbeConcurrency int
	// VerifyConcurrency bounds the yt-dlp stage, which is far heavier.
	VerifyConcurrency int
	// Client is used for the probe stage.
	Client *http.Client
	// Progress, if set, is called as each track is resolved.
	Progress func(done, total int)
}

func NewChecker() *Checker {
	return &Checker{
		ProbeConcurrency:  10,
		VerifyConcurrency: 4,
		Client:            &http.Client{Timeout: 15 * time.Second},
	}
}

// Check resolves every track, returning one Result per input track.
func (c *Checker) Check(tracks []Track) []Result {
	results := make([]Result, len(tracks))
	var done int
	var mu sync.Mutex

	tick := func() {
		mu.Lock()
		done++
		n := done
		mu.Unlock()
		if c.Progress != nil {
			c.Progress(n, len(tracks))
		}
	}

	// Stage 1 — probe. 200 settles a track as alive; anything else is a
	// candidate for the authoritative stage.
	type candidate struct{ idx int }
	var candidates []candidate
	var candMu sync.Mutex

	c.each(len(tracks), c.ProbeConcurrency, func(i int) {
		t := tracks[i]
		results[i] = Result{YoutubeID: t.YoutubeID, Title: t.Title}
		if c.probe(t.YoutubeID) == http.StatusOK {
			results[i].Status = StatusAlive
			tick()
			return
		}
		candMu.Lock()
		candidates = append(candidates, candidate{i})
		candMu.Unlock()
	})

	// Stage 2 — verify candidates with yt-dlp.
	c.each(len(candidates), c.VerifyConcurrency, func(k int) {
		i := candidates[k].idx
		status, reason := c.verify(results[i].YoutubeID)
		results[i].Status = status
		results[i].Reason = reason
		tick()
	})

	return results
}

// probe asks the oEmbed endpoint about one video. Any transport error returns
// 0, which is not 200, so the track falls through to yt-dlp rather than being
// judged here.
func (c *Checker) probe(youtubeID string) int {
	endpoint := "https://www.youtube.com/oembed?url=" +
		url.QueryEscape("https://www.youtube.com/watch?v="+youtubeID) + "&format=json"
	resp, err := c.Client.Get(endpoint)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// verify asks yt-dlp to resolve the video without downloading it. Success means
// the track plays; failure is classified, and only a recognised
// gone-for-everyone error counts as dead.
func (c *Checker) verify(youtubeID string) (Status, string) {
	cmd := exec.Command("yt-dlp", "--simulate", "--no-warnings",
		"--print", "%(title)s",
		"https://www.youtube.com/watch?v="+youtubeID)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return StatusAlive, ""
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return StatusUnknown, fmt.Sprintf("yt-dlp failed: %v", err)
	}
	// Use the last ERROR line; yt-dlp may print several.
	lines := strings.Split(text, "\n")
	errLine := lines[len(lines)-1]
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "ERROR:") {
			errLine = strings.TrimSpace(l)
		}
	}
	return classify(errLine)
}

// each runs fn over indices [0,n) with at most workers in flight.
func (c *Checker) each(n, workers int, fn func(i int)) {
	if n == 0 {
		return
	}
	if workers < 1 {
		workers = 1
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			fn(i)
		}(i)
	}
	wg.Wait()
}
