package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gitcoder89431/tui-tube/internal/app"
	"github.com/gitcoder89431/tui-tube/internal/db"
	"github.com/gitcoder89431/tui-tube/internal/enrich"
	"github.com/gitcoder89431/tui-tube/internal/mcpserver"
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
	return filepath.Join(home, ".local", "share", "tui-tube", "tuitube.db")
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
	case "enrich":
		runEnrich(*dbPath, args[1:])
	case "bootstrap":
		runBootstrap(*dbPath, args[1:])
	case "mcp":
		runMCP(*dbPath)
	case "":
		runTUI(*dbPath)
	default:
		fmt.Fprintf(os.Stderr, "tuitube: unknown subcommand %q\n", subcommand)
		fmt.Fprintln(os.Stderr, "usage: tuitube [sync | add-station | clean | bootstrap | mcp]")
		os.Exit(1)
	}
}

func runEnrich(dbPath string, args []string) {
	fs := flag.NewFlagSet("enrich", flag.ExitOnError)
	apiKey := fs.String("key", os.Getenv("LASTFM_KEY"), "Last.fm API key (or set LASTFM_KEY env var)")
	fs.Parse(args)

	if *apiKey == "" {
		fmt.Fprintln(os.Stderr, "tuitube enrich: --key or LASTFM_KEY required")
		fmt.Fprintln(os.Stderr, "  get a free key at: https://www.last.fm/api/account/create")
		os.Exit(1)
	}

	database, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube enrich: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := enrich.Run(database, *apiKey, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "tuitube enrich: %v\n", err)
		os.Exit(1)
	}
}

func runBootstrap(dbPath string, args []string) {
	fs := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	catalogPath := fs.String("catalog", db.DefaultCatalogPath(), "path to catalog.db")
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

	fmt.Printf("merging catalog from %s...\n", *catalogPath)
	if err := database.MergeCatalog(*catalogPath); err != nil {
		fmt.Fprintf(os.Stderr, "tuitube bootstrap: merge: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("done. run: tuitube")
}

// autoMerge checks if a catalog is available and newer than what's already
// merged, and silently merges it in. Called on every TUI/MCP startup.
func autoMerge(dbPath string) {
	catalogPath := db.DefaultCatalogPath()
	if _, err := os.Stat(catalogPath); err != nil {
		return // no catalog installed, skip
	}
	database, err := db.Open(dbPath)
	if err != nil {
		return
	}
	defer database.Close()

	cat, err := db.Open(catalogPath)
	if err != nil {
		return
	}
	catVersion := cat.CatalogVersion()
	cat.Close()

	if catVersion == "" || catVersion == database.CatalogVersion() {
		return // already up to date
	}

	_ = database.InitUserDB()
	_ = database.MergeCatalog(catalogPath)
}

func runTUI(dbPath string) {
	autoMerge(dbPath)
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
