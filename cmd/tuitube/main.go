package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gitcoder89431/tui-tube/internal/agentlog"
	"github.com/gitcoder89431/tui-tube/internal/app"
	"github.com/gitcoder89431/tui-tube/internal/db"
	"github.com/gitcoder89431/tui-tube/internal/mcpserver"
	"github.com/gitcoder89431/tui-tube/internal/player"
	tubesync "github.com/gitcoder89431/tui-tube/internal/sync"
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

global flags:
  --db PATH       path to user database (default: ~/.local/share/tuitube/tuitube.db)
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
		fmt.Printf(`{"playing":%v,"paused":%v,"youtube_id":%q,"title":%q,"artist":%q,"time_pos":%.1f,"duration":%.1f}`+"\n",
			s.Playing, s.Paused, s.YoutubeID, s.Title, s.Artist, tp, dur)
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
	fs.Parse(args)

	database, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube sync: open db: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	stations, err := database.ListStations()
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube sync: list stations: %v\n", err)
		os.Exit(1)
	}

	total := 0
	for _, s := range stations {
		if *stationID != "" && s.ID != *stationID {
			continue
		}
		fmt.Printf("[%s]\n", s.Name)
		n, err := tubesync.Station(database, s, os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  error: %v\n", err)
			continue
		}
		fmt.Printf("  +%d new tracks\n\n", n)
		total += n
	}

	if total > 0 {
		fmt.Println("rebuilding FTS index...")
		if err := database.RebuildFTS(); err != nil {
			fmt.Fprintf(os.Stderr, "fts rebuild: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Printf("done. %d new tracks total.\n", total)
}

func runClean(dbPath string) {
	database, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube clean: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

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
