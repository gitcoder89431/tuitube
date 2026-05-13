package app

import (
	"fmt"
	"sort"

	"github.com/gitcoder89431/tui-tube/internal/commands"
	"github.com/gitcoder89431/tui-tube/internal/db"
	"github.com/gitcoder89431/tui-tube/internal/debug"
	"github.com/gitcoder89431/tui-tube/internal/player"
	"github.com/gitcoder89431/tui-tube/internal/screens"
	"github.com/gitcoder89431/tui-tube/internal/theme"
	tea "charm.land/bubbletea/v2"
)

const defaultScreen = "library"

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

type Model struct {
	width  int
	height int

	activeScreen string
	screens      map[string]screens.Screen
	screenOrder  []string

	showSidebar        bool
	showHelp           bool
	showCommandPalette bool

	focus FocusArea
	keys  KeyMap

	commands       *commands.Registry
	commandPalette commands.PaletteModel

	theme    theme.Theme
	logs     *debug.Log
	meta     BuildInfo
	database *db.DB

	nowPlaying      *player.State
	timePos         float64
	duration        float64
	showVisualizer  bool
	visFrame        uint64
	queue       []db.Track
	queuePos    int
	downloading map[string]bool
	downloaded  map[string]bool
}

func New(meta BuildInfo, database *db.DB) Model {
	log := debug.NewLog()
	log.Info("App started")

	// ensure downloads table exists and load persisted state
	downloaded := make(map[string]bool)
	if database != nil {
		if err := database.InitDownloads(); err != nil {
			log.Error("init downloads", err)
		} else if dl, err := database.LoadDownloaded(); err != nil {
			log.Error("load downloads", err)
		} else {
			downloaded = dl
		}
	}

	m := Model{
		activeScreen: defaultScreen,
		screens:      make(map[string]screens.Screen),
		showSidebar:  true,
		focus:        FocusMain,
		keys:         DefaultKeyMap(),
		commands:     commands.NewRegistry(),
		theme:        theme.Phosphor(),
		logs:         log,
		meta:         meta,
		database:     database,
		downloading: make(map[string]bool),
		downloaded:  downloaded,
	}

	m.registerScreens()
	m.registerCommands()
	m.syncDownloadStateToLibrary()
	m.commandPalette = commands.NewPaletteModel(m.commands, theme.BuiltIns())
	return m
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		func() tea.Msg { return tea.RequestWindowSize() },
		nowPlayingTick(),
		sessionResumeCmd(),
	}
	for _, screen := range m.screens {
		cmds = append(cmds, screen.Init())
	}
	return tea.Batch(cmds...)
}

func (m *Model) registerScreens() {
	m.screens["library"] = screens.NewLibrary(m.database, m.theme)
	m.screens["playlists"] = screens.NewPlaylists(m.database, m.theme)
	m.screens["settings"] = screens.NewSettings(screens.SettingsState{
		ThemeName:      m.theme.Name,
		SidebarVisible: m.showSidebar,
		Version:        m.meta.Version,
		Commit:         m.meta.Commit,
		Date:           m.meta.Date,
	}, m.theme)
	m.screens["help"] = screens.NewHelp(m.keys.FullHelp(), m.theme)
	m.screens["logs"] = screens.NewLogs(m.logs)
	m.refreshScreenOrder()
}

// refreshScreenOrder rebuilds the sidebar ordering. Screen IDs must be unique.
// Add new primary screens to the preferred slice to control their position;
// all other registered screens are appended in sorted order after them.
func (m *Model) refreshScreenOrder() {
	m.screenOrder = m.screenOrder[:0]
	for id := range m.screens {
		m.screenOrder = append(m.screenOrder, id)
	}
	sort.Strings(m.screenOrder)
	preferred := []string{"library", "playlists", "settings", "help", "logs"}
	ordered := make([]string, 0, len(m.screenOrder))
	seen := make(map[string]bool)
	for _, id := range preferred {
		if _, ok := m.screens[id]; ok {
			ordered = append(ordered, id)
			seen[id] = true
		}
	}
	for _, id := range m.screenOrder {
		if !seen[id] {
			ordered = append(ordered, id)
		}
	}
	m.screenOrder = ordered
}

func (m *Model) registerCommands() {
	m.commands.Register(commands.Command{ID: "go-library", Module: commands.ModuleHome, Title: "Go to Library", Description: "Open the track library", Keywords: []string{"library", "tracks", "songs"}, Run: func() tea.Cmd { return func() tea.Msg { return routeMsg{"library"} } }})
	m.commands.Register(commands.Command{ID: "go-playlists", Module: commands.ModuleHome, Title: "Go to Playlists", Description: "Open the playlists lobby", Keywords: []string{"playlists", "stations", "channels"}, Run: func() tea.Cmd { return func() tea.Msg { return routeMsg{"playlists"} } }})
	m.commands.Register(commands.Command{ID: "go-settings", Module: commands.ModuleSettings, Title: "Go to Settings", Description: "Open application settings", Keywords: []string{"settings", "config"}, Run: func() tea.Cmd { return func() tea.Msg { return routeMsg{"settings"} } }})
	m.commands.Register(commands.Command{ID: "go-help", Module: commands.ModuleHelp, Title: "Go to Help", Description: "Open keyboard and command documentation", Keywords: []string{"help", "keys", "docs"}, Run: func() tea.Cmd { return func() tea.Msg { return routeMsg{"help"} } }})
	m.commands.Register(commands.Command{ID: "go-logs", Module: commands.ModuleLogs, Title: "Go to Logs", Description: "Open debug event log", Keywords: []string{"logs", "debug", "events"}, Run: func() tea.Cmd { return func() tea.Msg { return routeMsg{"logs"} } }})
	m.commands.Register(commands.Command{ID: "toggle-sidebar", Title: "Toggle Sidebar", Description: "Show or hide sidebar navigation", Keywords: []string{"sidebar", "layout"}, Run: func() tea.Cmd { return func() tea.Msg { return toggleSidebarMsg{} } }})
	m.commands.Register(commands.Command{ID: "themes", Title: "Themes", Description: "Preview and select a theme", Keywords: []string{"theme", "themes", "appearance", "colors", "dark", "muted", "phosphor", "miami"}, OpensPage: "themes"})
	m.commands.Register(commands.Command{ID: "quit", Title: "Quit", Description: "Exit the application", Keywords: []string{"exit", "close"}, Run: func() tea.Cmd { return func() tea.Msg { return quitMsg{} } }})
}

func (m *Model) switchScreen(id string) {
	if _, ok := m.screens[id]; !ok {
		m.logs.Warn(fmt.Sprintf("Unknown screen requested: %s", id))
		return
	}
	if m.activeScreen != id {
		m.activeScreen = id
		m.logs.Info(fmt.Sprintf("Screen changed to %s", id))
	}
}

func (m Model) CurrentScreenID() string { return m.activeScreen }

func (m Model) SwitchScreenForTest(id string) Model {
	m.switchScreen(id)
	return m
}
