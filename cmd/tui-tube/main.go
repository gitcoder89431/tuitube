package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gitcoder89431/tui-tube/internal/app"
	"github.com/gitcoder89431/tui-tube/internal/db"
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
		return "tui-tube.db"
	}
	return filepath.Join(home, ".local", "share", "tui-tube", "tui-tube.db")
}

func main() {
	showVersion := flag.Bool("version", false, "print version information")
	dbPath := flag.String("db", defaultDBPath(), "path to SQLite database")
	flag.Parse()

	if *showVersion {
		fmt.Printf("tui-tube %s (%s, %s)\n", version, commit, date)
		return
	}

	database, err := db.Open(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tui-tube: open db: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	meta := app.BuildInfo{Version: version, Commit: commit, Date: date}
	program := tea.NewProgram(app.New(meta, database))
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui-tube: %v\n", err)
		os.Exit(1)
	}
}
