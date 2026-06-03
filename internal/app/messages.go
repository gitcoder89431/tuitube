package app

import (
	tea "charm.land/bubbletea/v2"
	"github.com/gitcoder89431/tuitube/internal/db"
)

// routeMsg requests navigation to a registered screen ID.
// Use screen IDs defined in registerScreens — unknown IDs are logged and ignored.
type routeMsg struct{ ScreenID string }

// toggleSidebarMsg shows or hides the sidebar.
type toggleSidebarMsg struct{}

// quitMsg signals the application to exit.
type quitMsg struct{}

// commandsExecutedMsg carries the result of a command palette action.
// Title is shown in the header; Cmd is the tea.Cmd returned by the command's Run function.
type commandsExecutedMsg struct {
	Title string
	Cmd   tea.Cmd
}

// nowPlayingTickMsg fires every second to refresh the now-playing bar.
type nowPlayingTickMsg struct{}

// visTickMsg fires at ~30fps to advance the visualizer animation.
type visTickMsg struct{}

// playlistOpenMsg carries pre-loaded tracks from a playlist selection.
type playlistOpenMsg struct {
	title  string
	tracks []db.Track
}
