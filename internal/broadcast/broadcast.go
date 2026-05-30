// Package broadcast serves a playlist as a continuous HTTP audio stream.
// Only locally downloaded tracks are played — missing tracks are downloaded
// automatically before broadcasting begins.
package broadcast

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gitcoder89431/tuitube/internal/db"
)

// Server streams a playlist as an Icecast-compatible HTTP radio station.
type Server struct {
	database   *db.DB
	playlistID int64
	port       int
	lan        bool

	mu       sync.RWMutex
	clients  map[chan []byte]struct{}
	nowTitle string
	nowArtist string
}

func New(database *db.DB, playlistID int64, port int, lan bool) *Server {
	return &Server{
		database:   database,
		playlistID: playlistID,
		port:       port,
		lan:        lan,
		clients:    make(map[chan []byte]struct{}),
	}
}

// Prepare checks which tracks in the playlist are missing and downloads them.
// Prints progress to w. Returns the ordered list of local filepaths ready to broadcast.
func (s *Server) Prepare(w io.Writer) ([]db.Track, error) {
	tracks, err := s.database.ListPlaylistTracks(s.playlistID)
	if err != nil {
		return nil, fmt.Errorf("load playlist: %w", err)
	}
	if len(tracks) == 0 {
		return nil, fmt.Errorf("playlist is empty")
	}

	downloaded, err := s.database.LoadDownloaded()
	if err != nil {
		return nil, fmt.Errorf("load downloads: %w", err)
	}

	var missing []db.Track
	for _, t := range tracks {
		if downloaded[t.YoutubeID] == "" {
			missing = append(missing, t)
		}
	}

	if len(missing) > 0 {
		fmt.Fprintf(w, "→ %d/%d tracks need downloading\n", len(missing), len(tracks))
		dlPath, err := resolveDownloadPath()
		if err != nil {
			return nil, err
		}
		_ = os.MkdirAll(dlPath, 0755)

		for i, t := range missing {
			label := t.SongTitle
			if t.Artist != "" {
				label = t.Artist + " - " + t.SongTitle
			}
			fmt.Fprintf(w, "  [%d/%d] %s\n", i+1, len(missing), label)
			fp, err := downloadTrack(t, dlPath)
			if err != nil {
				fmt.Fprintf(w, "  ✗ failed: %v\n", err)
				continue
			}
			if err := s.database.MarkDownloaded(t.YoutubeID, fp); err != nil {
				fmt.Fprintf(w, "  ✗ db write failed: %v\n", err)
				continue
			}
			downloaded[t.YoutubeID] = fp
			fmt.Fprintf(w, "  ✓ done\n")
		}
	}

	// build final ordered list of downloaded tracks, skip any that failed
	var ready []db.Track
	for _, t := range tracks {
		if downloaded[t.YoutubeID] != "" {
			ready = append(ready, t)
		}
	}
	if len(ready) == 0 {
		return nil, fmt.Errorf("no tracks available after download attempt")
	}
	return ready, nil
}

// Serve starts the HTTP server and begins streaming the playlist on loop.
// Blocks until the server is stopped.
func (s *Server) Serve(tracks []db.Track, w io.Writer) error {
	bind := "127.0.0.1"
	if s.lan {
		bind = "0.0.0.0"
	}
	addr := fmt.Sprintf("%s:%d", bind, s.port)

	mux := http.NewServeMux()
	mux.HandleFunc("/stream", s.handleStream)
	mux.HandleFunc("/", s.handleNowPlaying)

	srv := &http.Server{Addr: addr, Handler: mux}

	fmt.Fprintf(w, "→ broadcasting on http://localhost:%d/stream\n", s.port)
	if s.lan {
		if ip := lanIP(); ip != "" {
			fmt.Fprintf(w, "→ LAN:       http://%s:%d/stream\n", ip, s.port)
		}
	}
	fmt.Fprintf(w, "→ now playing page: http://localhost:%d/\n\n", s.port)

	// start broadcast loop in background
	go s.broadcastLoop(tracks)

	return srv.ListenAndServe()
}

// broadcastLoop feeds each track into the broadcaster in playlist order, looping forever.
func (s *Server) broadcastLoop(tracks []db.Track) {
	downloaded, _ := s.database.LoadDownloaded()
	for {
		for _, t := range tracks {
			fp := downloaded[t.YoutubeID]
			if fp == "" {
				continue
			}
			s.mu.Lock()
			s.nowTitle = t.SongTitle
			s.nowArtist = t.Artist
			s.mu.Unlock()

			s.streamFile(fp)
		}
	}
}

// streamFile pipes one MP3 file through ffmpeg and fans the output to all connected clients.
func (s *Server) streamFile(fp string) {
	cmd := exec.Command("ffmpeg",
		"-re",            // realtime (don't process faster than playback speed)
		"-i", fp,
		"-vn",
		"-f", "mp3",
		"-q:a", "2",
		"pipe:1",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	if err := cmd.Start(); err != nil {
		return
	}
	defer cmd.Wait()

	buf := make([]byte, 4096)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			s.fanOut(chunk)
		}
		if err != nil {
			return
		}
	}
}

func (s *Server) fanOut(chunk []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for ch := range s.clients {
		select {
		case ch <- chunk:
		default: // slow client — drop chunk rather than block broadcaster
		}
	}
}

func (s *Server) addClient(ch chan []byte) {
	s.mu.Lock()
	s.clients[ch] = struct{}{}
	s.mu.Unlock()
}

func (s *Server) removeClient(ch chan []byte) {
	s.mu.Lock()
	delete(s.clients, ch)
	s.mu.Unlock()
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("icy-name", "tuitube")
	w.Header().Set("icy-genre", "mixed")
	w.Header().Set("icy-metaint", "8192")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	ch := make(chan []byte, 32)
	s.addClient(ch)
	defer s.removeClient(ch)

	bytesSinceMetadata := 0
	const metaInterval = 8192

	for {
		select {
		case <-r.Context().Done():
			return
		case chunk, ok := <-ch:
			if !ok {
				return
			}
			// inject ICY metadata at interval boundaries
			remaining := chunk
			for len(remaining) > 0 {
				space := metaInterval - bytesSinceMetadata
				if space > len(remaining) {
					space = len(remaining)
				}
				w.Write(remaining[:space])
				bytesSinceMetadata += space
				remaining = remaining[space:]

				if bytesSinceMetadata >= metaInterval {
					w.Write(s.icecastMetaBlock())
					bytesSinceMetadata = 0
				}
			}
			flusher.Flush()
		}
	}
}

func (s *Server) icecastMetaBlock() []byte {
	s.mu.RLock()
	title, artist := s.nowTitle, s.nowArtist
	s.mu.RUnlock()

	label := title
	if artist != "" {
		label = artist + " - " + title
	}
	meta := fmt.Sprintf("StreamTitle='%s';", strings.ReplaceAll(label, "'", " "))
	// ICY metadata block: 1 byte length (in 16-byte units), then the string padded to that length
	blocks := (len(meta) + 15) / 16
	padded := make([]byte, 1+blocks*16)
	padded[0] = byte(blocks)
	copy(padded[1:], meta)
	return padded
}

func (s *Server) handleNowPlaying(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	title, artist := s.nowTitle, s.nowArtist
	s.mu.RUnlock()
	label := title
	if artist != "" {
		label = artist + " — " + title
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!doctype html><html><head>
<meta charset=utf-8>
<meta http-equiv="refresh" content="10">
<title>tuitube radio</title>
<style>body{font-family:monospace;background:#0d0d0d;color:#ccc;display:flex;align-items:center;justify-content:center;height:100vh;margin:0}
.card{text-align:center}.title{font-size:1.4em;color:#fff;margin-bottom:.5em}.stream{color:#888;font-size:.85em}</style>
</head><body><div class="card">
<div style="color:#888;margin-bottom:1em">▶ tuitube radio</div>
<div class="title">%s</div>
<div class="stream">http://%s/stream</div>
</div></body></html>`, label, w.Header().Get("Host"))
}

func downloadTrack(t db.Track, dlPath string) (string, error) {
	fp := filepath.Join(dlPath, sanitizeFilename(t.Artist, t.SongTitle)+".mp3")
	url := "https://www.youtube.com/watch?v=" + t.YoutubeID
	cmd := exec.Command("yt-dlp", "-x", "--audio-format", "mp3", "-o", fp, url)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return fp, nil
}

func resolveDownloadPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, "Music", "tuitube"), nil
}

func sanitizeFilename(artist, title string) string {
	unsafe := `/\:*?"<>|`
	clean := func(s string) string {
		var out strings.Builder
		for _, r := range s {
			if strings.ContainsRune(unsafe, r) {
				out.WriteRune('-')
			} else {
				out.WriteRune(r)
			}
		}
		return strings.TrimSpace(out.String())
	}
	if artist != "" {
		return clean(artist) + " - " + clean(title)
	}
	return clean(title)
}

func lanIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}
			return ip.String()
		}
	}
	return ""
}

