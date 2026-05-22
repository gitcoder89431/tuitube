package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gitcoder89431/tuitube/internal/agentlog"
	"github.com/gitcoder89431/tuitube/internal/app"
	"github.com/gitcoder89431/tuitube/internal/broadcast"
	"github.com/gitcoder89431/tuitube/internal/db"
	"github.com/gitcoder89431/tuitube/internal/mcpserver"
	"github.com/gitcoder89431/tuitube/internal/player"
	tubesync "github.com/gitcoder89431/tuitube/internal/sync"
	tea "charm.land/bubbletea/v2"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func defaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "tuitube.db"
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "tuitube", "tuitube.db")
	}
	return filepath.Join(home, ".local", "share", "tuitube", "tuitube.db")
}

func main() {
	// global flags (must come before subcommand)
	dbPath := flag.String("db", defaultDBPath(), "path to SQLite database")
	showVersion := flag.Bool("version", false, "print version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("tuitube %s (%s, %s)\n", version, commit, date)
		return
	}

	args := flag.Args()
	subcommand := ""
	if len(args) > 0 {
		subcommand = args[0]
	}

	switch subcommand {
	case "sync":
		runSync(*dbPath, args[1:])
	case "add-station":
		runAddStation(*dbPath, args[1:])
	case "clean":
		runClean(*dbPath)
	case "status":
		runStatus(*dbPath, args[1:])
	case "doctor":
		runDoctor(*dbPath)
	case "bootstrap":
		runBootstrap(*dbPath, args[1:])
	case "mcp":
		runMCP(*dbPath)
	case "serve":
		runServe(*dbPath, args[1:])
	case "":
		runTUI(*dbPath)
	default:
		fmt.Fprintf(os.Stderr, "tuitube: unknown subcommand %q\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `usage: tuitube [--db PATH] <subcommand>

subcommands:
  (none)          launch the TUI
  status          show now-playing state (--json for machine-readable)
  sync            pull new tracks from YouTube channels (--station ID for one)
  add-station     add a YouTube channel (--url URL --name NAME)
  bootstrap       merge catalog.db into user DB (--catalog PATH)
  clean           normalise track titles in the DB
  doctor          check mpv, yt-dlp, and DB health
  mcp             start the MCP server (for Claude Code integration)
  serve           broadcast a playlist as an HTTP radio stream (--playlist ID)

global flags:
  --db PATH       path to user database (default: platform-specific data dir)
  --version       print version`)
}

func runStatus(dbPath string, args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "output as JSON")
	fs.Parse(args)

	s := player.NowPlaying()
	if s == nil {
		if *asJSON {
			fmt.Println(`{"playing":false}`)
		} else {
			fmt.Println("nothing playing")
		}
		return
	}

	tp, dur := player.Progress()
	if *asJSON {
		pct := 0.0
		if dur > 0 {
			pct = tp / dur * 100
		}
		fmtTime := func(sec float64) string {
			s := int(sec)
			return fmt.Sprintf("%d:%02d", s/60, s%60)
		}
		thumbnail := ""
		if db, err := db.Open(dbPath); err == nil {
			var t string
			_ = db.QueryRow("SELECT COALESCE(thumbnail,'') FROM tracks WHERE youtube_id=?", s.YoutubeID).Scan(&t)
			thumbnail = t
			db.Close()
		}
		fmt.Printf(`{"playing":%v,"paused":%v,"youtube_id":%q,"title":%q,"artist":%q,"thumbnail":%q,"time_pos":%.1f,"duration":%.1f,"progress_pct":%.1f,"time_fmt":%q,"duration_fmt":%q}`+"\n",
			s.Playing, s.Paused, s.YoutubeID, s.Title, s.Artist, thumbnail, tp, dur, pct, fmtTime(tp), fmtTime(dur))
		return
	}

	icon := "▶"
	if s.Paused {
		icon = "⏸"
	}
	label := s.Title
	if s.Artist != "" {
		label = s.Artist + " — " + s.Title
	}
	progress := ""
	if dur > 0 {
		progress = fmt.Sprintf("  [%d:%02d / %d:%02d]", int(tp)/60, int(tp)%60, int(dur)/60, int(dur)%60)
	}
	fmt.Printf("%s %s%s\n", icon, label, progress)
}

func runDoctor(dbPath string) {
	ok := true
	check := func(label, cmd string) {
		_, err := exec.LookPath(cmd)
		if err != nil {
			fmt.Printf("  ✗ %s: not found in PATH\n", label)
			ok = false
		} else {
			fmt.Printf("  ✓ %s\n", label)
		}
	}

	fmt.Println("dependencies:")
	check("mpv", "mpv")
	check("yt-dlp", "yt-dlp")

	fmt.Println("\ndatabase:")
	database, err := db.Open(dbPath)
	if err != nil {
		fmt.Printf("  ✗ cannot open %s: %v\n", dbPath, err)
		ok = false
	} else {
		defer database.Close()
		var trackCount int
		if err := database.QueryRow("SELECT COUNT(*) FROM tracks").Scan(&trackCount); err != nil {
			fmt.Printf("  ✗ tracks table: %v\n", err)
			ok = false
		} else {
			fmt.Printf("  ✓ %s (%d tracks)\n", dbPath, trackCount)
		}
		var ftsOk int
		if err := database.QueryRow("SELECT COUNT(*) FROM tracks_fts").Scan(&ftsOk); err != nil {
			fmt.Printf("  ✗ FTS index: %v\n", err)
			ok = false
		} else {
			fmt.Println("  ✓ FTS index")
		}
	}

	fmt.Println()
	if ok {
		fmt.Println("all checks passed")
	} else {
		fmt.Fprintln(os.Stderr, "some checks failed")
		os.Exit(1)
	}
}

func runBootstrap(dbPath string, args []string) {
	fs := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	catalogPath := fs.String("catalog", db.DefaultCatalogPath(), "path to catalog.db")
	dryRun := fs.Bool("dry-run", false, "show what would be merged without writing")
	fs.Parse(args)

	if _, err := os.Stat(*catalogPath); err != nil {
		fmt.Fprintf(os.Stderr, "tuitube bootstrap: catalog not found at %s\n", *catalogPath)
		fmt.Fprintln(os.Stderr, "  use --catalog /path/to/catalog.db to specify location")
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "tuitube bootstrap: %v\n", err)
		os.Exit(1)
	}

	database, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube bootstrap: open db: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := database.InitUserDB(); err != nil {
		fmt.Fprintf(os.Stderr, "tuitube bootstrap: init: %v\n", err)
		os.Exit(1)
	}

	if *dryRun {
		fmt.Printf("dry-run — checking catalog %s...\n", *catalogPath)
	} else {
		fmt.Printf("merging catalog from %s...\n", *catalogPath)
	}
	result, err := database.MergeCatalogFull(*catalogPath, *dryRun)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube bootstrap: merge: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("catalog version : %s\n", result.CatalogVersion)
	fmt.Printf("new stations    : %d\n", result.NewStations)
	fmt.Printf("new tracks      : %d\n", result.NewTracks)
	fmt.Printf("new playlists   : %d\n", result.NewPlaylists)
	if !*dryRun {
		fmt.Println("done. run: tuitube")
	}
}

// autoMerge checks if a catalog is available and newer than what's already
// merged, and silently merges it in. Called on every TUI/MCP startup.
func autoMerge(dbPath string) {
	catalogPath := db.DefaultCatalogPath()
	if _, err := os.Stat(catalogPath); err != nil {
		return // no catalog installed, skip
	}
	cat, err := db.Open(catalogPath)
	if err != nil {
		return
	}
	catVersion := cat.CatalogVersion()
	cat.Close()
	if catVersion == "" {
		return
	}

	userDB, err := db.Open(dbPath)
	if err != nil {
		return
	}
	defer userDB.Close()

	if catVersion == userDB.CatalogVersion() {
		return // already up to date
	}

	_ = userDB.InitUserDB()
	_ = userDB.MergeCatalog(catalogPath)
}

func runTUI(dbPath string) {
	agentlog.Clear() // fresh log each TUI session
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "tuitube: %v\n", err)
		os.Exit(1)
	}
	database, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube: open db: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()
	// ensure tables exist — covers first run before bootstrap
	if err := database.InitUserDB(); err != nil {
		fmt.Fprintf(os.Stderr, "tuitube: init db: %v\n", err)
		os.Exit(1)
	}
	autoMerge(dbPath)

	meta := app.BuildInfo{Version: version, Commit: commit, Date: date}
	program := tea.NewProgram(app.New(meta, database))
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tuitube: %v\n", err)
		os.Exit(1)
	}
}

func runSync(dbPath string, args []string) {
	fs := flag.NewFlagSet("sync", flag.ExitOnError)
	stationID := fs.String("station", "", "sync only this station ID (default: all)")
	asJSON := fs.Bool("json", false, "output machine-readable JSON summary")
	fs.Parse(args)

	database, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube sync: open db: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	if !database.IsInitialized() {
		fmt.Fprintln(os.Stderr, "tuitube sync: database not initialized — run `tuitube bootstrap` first")
		os.Exit(1)
	}

	stations, err := database.ListStations()
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube sync: list stations: %v\n", err)
		os.Exit(1)
	}

	type stationResult struct {
		Station  string `json:"station"`
		Inserted int    `json:"inserted"`
		Error    string `json:"error,omitempty"`
	}
	var results []stationResult
	total := 0

	for _, s := range stations {
		if *stationID != "" && s.ID != *stationID {
			continue
		}
		var w io.Writer = os.Stdout
		if *asJSON {
			w = io.Discard
		} else {
			fmt.Printf("[%s]\n", s.Name)
		}
		n, err := tubesync.Station(database, s, w)
		r := stationResult{Station: s.Name, Inserted: n}
		if err != nil {
			r.Error = err.Error()
			if !*asJSON {
				fmt.Fprintf(os.Stderr, "  error: %v\n", err)
			}
		} else if !*asJSON {
			fmt.Printf("  +%d new tracks\n\n", n)
		}
		results = append(results, r)
		total += n
	}

	if total > 0 {
		if !*asJSON {
			fmt.Println("rebuilding FTS index...")
		}
		if err := database.RebuildFTS(); err != nil {
			fmt.Fprintf(os.Stderr, "fts rebuild: %v\n", err)
			os.Exit(1)
		}
	}

	if *asJSON {
		type syncSummary struct {
			StationsSynced int             `json:"stations_synced"`
			TracksAdded    int             `json:"tracks_added"`
			Stations       []stationResult `json:"stations"`
		}
		out, _ := json.Marshal(syncSummary{
			StationsSynced: len(results),
			TracksAdded:    total,
			Stations:       results,
		})
		fmt.Println(string(out))
	} else {
		fmt.Printf("done. %d new tracks total.\n", total)
	}
}

func runClean(dbPath string) {
	database, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube clean: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	if !database.IsInitialized() {
		fmt.Fprintln(os.Stderr, "tuitube clean: database not initialized — run `tuitube bootstrap` first")
		os.Exit(1)
	}

	tracks, err := database.AllTracks()
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube clean: %v\n", err)
		os.Exit(1)
	}

	updated := 0
	for _, t := range tracks {
		cleanedTitle := tubesync.CleanTitle(t.SongTitle)
		cleanedArtist := strings.TrimSpace(t.Artist)
		if cleanedTitle == t.SongTitle && cleanedArtist == t.Artist {
			continue
		}
		if err := database.UpdateTrackTitles(t.ID, cleanedTitle, cleanedArtist); err != nil {
			fmt.Fprintf(os.Stderr, "  error updating %s: %v\n", t.ID, err)
			continue
		}
		updated++
	}

	if updated > 0 {
		fmt.Printf("cleaned %d tracks, rebuilding FTS...\n", updated)
		if err := database.RebuildFTS(); err != nil {
			fmt.Fprintf(os.Stderr, "fts rebuild: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Printf("done. %d of %d tracks updated.\n", updated, len(tracks))
}

func runMCP(dbPath string) {
	database, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube mcp: open db: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()
	if err := mcpserver.New(database).Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "tuitube mcp: %v\n", err)
		os.Exit(1)
	}
}

func runAddStation(dbPath string, args []string) {
	fs := flag.NewFlagSet("add-station", flag.ExitOnError)
	channelURL := fs.String("url", "", "YouTube channel URL (required)")
	name := fs.String("name", "", "display name for the station (required)")
	fs.Parse(args)

	if *channelURL == "" || *name == "" {
		fmt.Fprintln(os.Stderr, "usage: tuitube add-station --url <channel-url> --name <name>")
		os.Exit(1)
	}

	database, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube add-station: open db: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	station, err := tubesync.DiscoverStation(*channelURL, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube add-station: %v\n", err)
		os.Exit(1)
	}
	station.Name = *name

	if err := database.UpsertStation(station); err != nil {
		fmt.Fprintf(os.Stderr, "tuitube add-station: upsert: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("added station %q (id=%s, uploads=%s)\n", station.Name, station.ID, station.UploadsPlaylistID)
	fmt.Printf("run: tuitube sync --station %s\n", station.ID)
}

func runServe(dbPath string, args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	playlistID := fs.Int64("playlist", 0, "playlist ID to broadcast (required)")
	port := fs.Int("port", 8080, "port to listen on")
	lan := fs.Bool("lan", false, "accept connections from LAN (non-loopback)")
	fs.Parse(args)

	if *playlistID == 0 {
		fmt.Fprintln(os.Stderr, "usage: tuitube serve --playlist <id> [--port 8080] [--lan]")
		fmt.Fprintln(os.Stderr, "  use: tuitube status or the TUI playlists screen to find playlist IDs")
		os.Exit(1)
	}

	database, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube serve: open db: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	srv := broadcast.New(database, *playlistID, *port, *lan)

	fmt.Println("→ checking playlist tracks...")
	tracks, err := srv.Prepare(os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube serve: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("→ %d tracks ready\n", len(tracks))

	if err := srv.Serve(tracks, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "tuitube serve: %v\n", err)
		os.Exit(1)
	}
}
