package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/gitcoder89431/tuitube/internal/agentlog"
	"github.com/gitcoder89431/tuitube/internal/app"
	"github.com/gitcoder89431/tuitube/internal/db"
	"github.com/gitcoder89431/tuitube/internal/dedupe"
	"github.com/gitcoder89431/tuitube/internal/links"
	"github.com/gitcoder89431/tuitube/internal/mcpserver"
	"github.com/gitcoder89431/tuitube/internal/player"
	tubesync "github.com/gitcoder89431/tuitube/internal/sync"
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
	case "check-links":
		runCheckLinks(*dbPath, args[1:])
	case "dedupe":
		runDedupe(*dbPath, args[1:])
	case "pruned":
		runPruned(*dbPath, args[1:])
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
  check-links     find dead YouTube links (--prune to remove them)
  dedupe          find duplicate songs across channels (--prune to remove)
  pruned          list tracks held back from sync (--forget to restore)
  mcp             start the MCP server (for Claude Code integration)

global flags:
  --db PATH       path to user database (default: platform-specific data dir)
  --version       print version`)
}

func runStatus(dbPath string, args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "output as JSON")
	fs.Parse(args)

	p := player.New(player.DefaultConfig())
	ctx := context.Background()
	s := p.NowPlaying()
	if s == nil {
		if *asJSON {
			fmt.Println(`{"playing":false}`)
		} else {
			fmt.Println("nothing playing")
		}
		return
	}

	tp, dur := p.Progress(ctx)
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
		if database, err := db.Open(dbPath); err == nil {
			defer database.Close()
			var t string
			_ = database.QueryRow("SELECT COALESCE(thumbnail,'') FROM tracks WHERE youtube_id=?", s.YoutubeID).Scan(&t)
			thumbnail = t
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

	p := player.New(player.DefaultConfig())
	meta := app.BuildInfo{Version: version, Commit: commit, Date: date}
	program := tea.NewProgram(app.New(meta, database, p))
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

	if err := database.InitUserDB(); err != nil {
		fmt.Fprintf(os.Stderr, "tuitube sync: migrate db: %v\n", err)
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
		// Re-derive from raw_title rather than re-cleaning the stored values:
		// rows whose artist failed to parse (en-dash separators, "Artist- Song")
		// only recover if the split runs again from the original title.
		cleanedArtist, cleanedTitle := tubesync.SplitArtistTitle(tubesync.CleanTitle(t.RawTitle))
		if cleanedArtist == "" {
			cleanedArtist = strings.TrimSpace(t.Artist)
		}
		cleanedArtist = tubesync.CleanArtist(cleanedArtist)
		if cleanedTitle == "" {
			cleanedTitle = tubesync.CleanTitle(t.SongTitle)
		}
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
	if err := mcpserver.New(database, player.New(player.DefaultConfig())).Serve(); err != nil {
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

func runCheckLinks(dbPath string, args []string) {
	fs := flag.NewFlagSet("check-links", flag.ExitOnError)
	prune := fs.Bool("prune", false, "delete tracks confirmed dead (default: report only)")
	asJSON := fs.Bool("json", false, "output machine-readable JSON summary")
	stationID := fs.String("station", "", "check only this station ID (default: all)")
	concurrency := fs.Int("concurrency", 10, "parallel probe requests")
	fs.Parse(args)

	database, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube check-links: open db: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	if !database.IsInitialized() {
		fmt.Fprintln(os.Stderr, "tuitube check-links: database not initialized — run `tuitube bootstrap` first")
		os.Exit(1)
	}

	var tracks []db.Track
	if *stationID != "" {
		tracks, err = database.ListStationTracks(*stationID)
	} else {
		tracks, err = database.AllTracks()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube check-links: load tracks: %v\n", err)
		os.Exit(1)
	}
	if len(tracks) == 0 {
		fmt.Fprintln(os.Stderr, "tuitube check-links: no tracks to check")
		return
	}

	input := make([]links.Track, len(tracks))
	for i, t := range tracks {
		input[i] = links.Track{
			YoutubeID: t.YoutubeID,
			Title:     strings.TrimSpace(t.Artist + " — " + t.SongTitle),
		}
	}

	checker := links.NewChecker()
	checker.ProbeConcurrency = *concurrency
	if !*asJSON {
		fmt.Fprintf(os.Stderr, "checking %d tracks...\n", len(input))
		checker.Progress = func(done, total int) {
			if done%500 == 0 || done == total {
				fmt.Fprintf(os.Stderr, "  %d/%d\n", done, total)
			}
		}
	}

	results := checker.Check(input)

	var dead, unknown []links.Result
	alive := 0
	for _, r := range results {
		switch r.Status {
		case links.StatusAlive:
			alive++
		case links.StatusDead:
			dead = append(dead, r)
		default:
			unknown = append(unknown, r)
		}
	}

	pruned := 0
	if *prune && len(dead) > 0 {
		ids := make([]string, len(dead))
		for i, r := range dead {
			ids[i] = r.YoutubeID
		}
		pruned, err = database.DeleteTracksByYoutubeID(ids)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tuitube check-links: prune: %v\n", err)
			os.Exit(1)
		}
		// Remember the removal so the next sync does not re-add them.
		if err := database.AddTombstones(ids, db.PruneReasonDead, nil); err != nil {
			fmt.Fprintf(os.Stderr, "tuitube check-links: record pruned: %v\n", err)
			os.Exit(1)
		}
		// tracks_fts is external-content with no triggers; without this it
		// would keep serving the tracks we just removed.
		if err := database.RebuildFTS(); err != nil {
			fmt.Fprintf(os.Stderr, "tuitube check-links: fts rebuild: %v\n", err)
			os.Exit(1)
		}
	}

	if *asJSON {
		out := struct {
			Checked int            `json:"checked"`
			Alive   int            `json:"alive"`
			Dead    []links.Result `json:"dead"`
			Unknown []links.Result `json:"unknown"`
			Pruned  int            `json:"pruned"`
		}{len(results), alive, dead, unknown, pruned}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out)
		return
	}

	fmt.Printf("\nchecked %d tracks: %d alive, %d dead, %d unknown\n",
		len(results), alive, len(dead), len(unknown))

	if len(dead) > 0 {
		fmt.Println("\ndead:")
		for _, r := range dead {
			fmt.Printf("  %s  %s (%s)\n", r.YoutubeID, r.Title, r.Reason)
		}
	}
	if len(unknown) > 0 {
		fmt.Println("\nunknown (not pruned — transient failure or region block):")
		for _, r := range unknown {
			fmt.Printf("  %s  %s (%s)\n", r.YoutubeID, r.Title, r.Reason)
		}
	}

	switch {
	case *prune:
		fmt.Printf("\npruned %d tracks, FTS rebuilt.\n", pruned)
	case len(dead) > 0:
		fmt.Printf("\nrun `tuitube check-links --prune` to remove %d dead tracks.\n", len(dead))
	default:
		fmt.Println("\nno dead links.")
	}
}

func runDedupe(dbPath string, args []string) {
	fs := flag.NewFlagSet("dedupe", flag.ExitOnError)
	prune := fs.Bool("prune", false, "delete redundant copies (default: report only)")
	asJSON := fs.Bool("json", false, "output machine-readable JSON summary")
	stationID := fs.String("station", "", "check only this station ID (default: all)")
	limit := fs.Int("limit", 25, "groups to list in the report (0 for all)")
	keepVersions := fs.Bool("keep-versions", false,
		"treat remixes, slowed/sped-up edits and guest-featuring cuts as distinct songs")
	fs.Parse(args)

	database, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube dedupe: open db: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	if !database.IsInitialized() {
		fmt.Fprintln(os.Stderr, "tuitube dedupe: database not initialized — run `tuitube bootstrap` first")
		os.Exit(1)
	}

	rows, err := database.DedupeCandidates(*stationID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube dedupe: load tracks: %v\n", err)
		os.Exit(1)
	}

	cands := make([]dedupe.Candidate, len(rows))
	for i, r := range rows {
		cands[i] = dedupe.Candidate{
			YoutubeID:   r.YoutubeID,
			Artist:      r.Artist,
			SongTitle:   r.SongTitle,
			RawTitle:    r.RawTitle,
			PublishedAt: r.PublishedAt,
			Station:     r.Station,
		}
	}

	groups := dedupe.Find(cands, dedupe.Options{KeepVersions: *keepVersions})
	redundant := 0
	var dropIDs []string
	remap := map[string]string{}
	for _, g := range groups {
		redundant += len(g.Drop)
		for _, d := range g.Drop {
			dropIDs = append(dropIDs, d.YoutubeID)
			remap[d.YoutubeID] = g.Keep.YoutubeID
		}
	}

	pruned, moved := 0, 0
	if *prune && len(dropIDs) > 0 {
		// Move playlist entries onto the keeper first, or deleting the
		// duplicate would quietly drop the song from curated playlists.
		moved, err = database.RemapPlaylistTracks(remap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tuitube dedupe: remap playlists: %v\n", err)
			os.Exit(1)
		}
		pruned, err = database.DeleteTracksByYoutubeID(dropIDs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tuitube dedupe: prune: %v\n", err)
			os.Exit(1)
		}
		// Remember which copy superseded each dropped one, so the next sync
		// does not re-add the duplicates.
		if err := database.AddTombstones(dropIDs, db.PruneReasonDuplicate, remap); err != nil {
			fmt.Fprintf(os.Stderr, "tuitube dedupe: record pruned: %v\n", err)
			os.Exit(1)
		}
		if err := database.RebuildFTS(); err != nil {
			fmt.Fprintf(os.Stderr, "tuitube dedupe: fts rebuild: %v\n", err)
			os.Exit(1)
		}
	}

	if *asJSON {
		type jsonCopy struct {
			YoutubeID string `json:"youtube_id"`
			RawTitle  string `json:"raw_title"`
			Station   string `json:"station"`
		}
		type jsonGroup struct {
			Artist string     `json:"artist"`
			Title  string     `json:"title"`
			Keep   jsonCopy   `json:"keep"`
			Drop   []jsonCopy `json:"drop"`
		}
		out := struct {
			Scanned   int         `json:"scanned"`
			Groups    []jsonGroup `json:"groups"`
			Redundant int         `json:"redundant"`
			Pruned    int         `json:"pruned"`
			Remapped  int         `json:"playlist_entries_remapped"`
		}{Scanned: len(cands), Redundant: redundant, Pruned: pruned, Remapped: moved}
		for _, g := range groups {
			jg := jsonGroup{
				Artist: g.Keep.Artist,
				Title:  g.Keep.SongTitle,
				Keep:   jsonCopy{g.Keep.YoutubeID, g.Keep.RawTitle, g.Keep.Station},
			}
			for _, d := range g.Drop {
				jg.Drop = append(jg.Drop, jsonCopy{d.YoutubeID, d.RawTitle, d.Station})
			}
			out.Groups = append(out.Groups, jg)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out)
		return
	}

	fmt.Printf("scanned %d tracks: %d duplicate groups, %d redundant copies\n",
		len(cands), len(groups), redundant)

	shown := groups
	if *limit > 0 && len(shown) > *limit {
		shown = shown[:*limit]
	}
	for _, g := range shown {
		fmt.Printf("\n  %s — %s\n", g.Keep.Artist, g.Keep.SongTitle)
		fmt.Printf("    keep  %s  %s  [%s]\n", g.Keep.YoutubeID, g.Keep.RawTitle, g.Keep.Station)
		for _, d := range g.Drop {
			fmt.Printf("    drop  %s  %s  [%s]\n", d.YoutubeID, d.RawTitle, d.Station)
		}
	}
	if *limit > 0 && len(groups) > *limit {
		fmt.Printf("\n  ... and %d more groups (--limit 0 to list all)\n", len(groups)-*limit)
	}

	switch {
	case *prune:
		fmt.Printf("\npruned %d tracks, %d playlist entries remapped, FTS rebuilt.\n", pruned, moved)
	case redundant > 0:
		fmt.Printf("\nrun `tuitube dedupe --prune` to remove %d redundant copies.\n", redundant)
	default:
		fmt.Println("\nno duplicates.")
	}
}

func runPruned(dbPath string, args []string) {
	fs := flag.NewFlagSet("pruned", flag.ExitOnError)
	reason := fs.String("reason", "", "filter by reason: dead or duplicate")
	forget := fs.String("forget", "", "comma-separated youtube_ids to un-prune")
	forgetAll := fs.Bool("forget-all", false, "un-prune everything")
	asJSON := fs.Bool("json", false, "output machine-readable JSON")
	limit := fs.Int("limit", 20, "entries to list (0 for all)")
	fs.Parse(args)

	database, err := db.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube pruned: open db: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := database.InitUserDB(); err != nil {
		fmt.Fprintf(os.Stderr, "tuitube pruned: migrate db: %v\n", err)
		os.Exit(1)
	}

	switch {
	case *forgetAll:
		n, err := database.ForgetTombstones(nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tuitube pruned: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("forgot %d entries. the next sync may re-add them.\n", n)
		return
	case *forget != "":
		var ids []string
		for _, id := range strings.Split(*forget, ",") {
			if id = strings.TrimSpace(id); id != "" {
				ids = append(ids, id)
			}
		}
		n, err := database.ForgetTombstones(ids)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tuitube pruned: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("forgot %d of %d entries. the next sync may re-add them.\n", n, len(ids))
		return
	}

	counts, err := database.CountTombstones()
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube pruned: %v\n", err)
		os.Exit(1)
	}
	entries, err := database.ListTombstones(*reason, *limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tuitube pruned: %v\n", err)
		os.Exit(1)
	}

	if *asJSON {
		type jsonEntry struct {
			YoutubeID  string `json:"youtube_id"`
			Reason     string `json:"reason"`
			ReplacedBy string `json:"replaced_by,omitempty"`
			PrunedAt   int64  `json:"pruned_at"`
		}
		out := struct {
			Counts  map[string]int `json:"counts"`
			Entries []jsonEntry    `json:"entries"`
		}{Counts: counts}
		for _, e := range entries {
			out.Entries = append(out.Entries, jsonEntry{e.YoutubeID, e.Reason, e.ReplacedBy, e.PrunedAt})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out)
		return
	}

	total := 0
	for _, n := range counts {
		total += n
	}
	if total == 0 {
		fmt.Println("nothing pruned. sync will add everything the channels list.")
		return
	}
	fmt.Printf("%d tracks pruned and held back from sync:\n", total)
	for r, n := range counts {
		fmt.Printf("  %-10s %d\n", r, n)
	}
	if len(entries) > 0 {
		fmt.Println()
		for _, e := range entries {
			if e.ReplacedBy != "" {
				fmt.Printf("  %s  %-10s replaced by %s\n", e.YoutubeID, e.Reason, e.ReplacedBy)
			} else {
				fmt.Printf("  %s  %s\n", e.YoutubeID, e.Reason)
			}
		}
		if *limit > 0 && total > len(entries) {
			fmt.Printf("\n  ... and %d more (--limit 0 to list all)\n", total-len(entries))
		}
	}
	fmt.Println("\nuse `tuitube pruned --forget ID` or `--forget-all` to let them back in.")
}
