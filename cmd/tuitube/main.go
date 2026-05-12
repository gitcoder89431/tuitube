package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gitcoder89431/tui-tube/internal/app"
	"github.com/gitcoder89431/tui-tube/internal/db"
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
	case "mcp":
		runMCP(*dbPath)
	case "":
		runTUI(*dbPath)
	default:
		fmt.Fprintf(os.Stderr, "tuitube: unknown subcommand %q\n", subcommand)
		fmt.Fprintln(os.Stderr, "usage: tuitube [sync | add-station | clean | mcp]")
		os.Exit(1)
	}
}

func runTUI(dbPath string) {
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
