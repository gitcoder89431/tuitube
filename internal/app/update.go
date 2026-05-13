package app

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gitcoder89431/tui-tube/internal/commands"
	"github.com/gitcoder89431/tui-tube/internal/player"
	"github.com/gitcoder89431/tui-tube/internal/screens"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/elpdev/tuimod"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// forward size to active screen so it can pre-compute layout if needed
		active := m.screens[m.activeScreen]
		updated, cmd := active.Update(msg)
		m.screens[m.activeScreen] = updated
		return m, cmd
	case routeMsg:
		m.switchScreen(msg.ScreenID)
		m.showCommandPalette = false
		m.updateDerivedScreens()
		return m, nil
	case toggleSidebarMsg:
		m.showSidebar = !m.showSidebar
		m.logs.Info(fmt.Sprintf("Sidebar toggled: %t", m.showSidebar))
		m.updateDerivedScreens()
		return m, nil
	case nowPlayingTickMsg:
		state := player.NowPlaying()
		if state != nil && state.Finished {
			return m, m.playNextInQueue()
		}
		m.nowPlaying = state
		m.syncNowPlayingToLibrary()
		return m, nowPlayingTick()
	case screens.BackMsg:
		m.switchScreen("playlists")
		if lib, ok := m.screens["library"].(screens.Library); ok {
			m.screens["library"] = lib.WithPlaylistTitle("")
			return m, lib.ReloadCmd()
		}
		return m, nil
	case screens.PlaylistsLoadedMsg:
		// route directly to playlists screen regardless of which screen is active
		if pl, ok := m.screens["playlists"].(screens.Playlists); ok {
			updated, cmd := pl.Update(msg)
			m.screens["playlists"] = updated
			return m, cmd
		}
		return m, nil
	case screens.PlaylistSelectedMsg:
		return m, func() tea.Msg {
			tracks, err := msg.Loader()
			if err != nil {
				return screens.TracksLoadedMsg{Err: err}
			}
			return playlistOpenMsg{title: msg.Title, tracks: tracks}
		}
	case playlistOpenMsg:
		m.switchScreen("library")
		if lib, ok := m.screens["library"].(screens.Library); ok {
			m.screens["library"] = lib.WithPlaylistTitle(msg.title).WithTracks(msg.tracks)
		}
		return m, nil
	case screens.TogglePauseMsg:
		if err := player.TogglePause(); err != nil {
			m.logs.Error("toggle pause", err)
		}
		m.nowPlaying = player.NowPlaying()
		return m, nil
	case screens.PlayTrackMsg:
		// same track → toggle pause instead of restarting
		if m.nowPlaying != nil && m.nowPlaying.YoutubeID == msg.YoutubeID {
			if err := player.TogglePause(); err != nil {
				m.logs.Error("toggle pause", err)
			}
		} else {
			if err := player.Play(msg.YoutubeID, msg.Title, msg.Artist); err != nil {
				m.logs.Error("mpv", err)
			}
			m.buildQueue(msg.YoutubeID)
		}
		m.nowPlaying = player.NowPlaying()
		m.syncNowPlayingToLibrary()
		return m, nil
	case screens.DownloadTrackMsg:
		m.downloading[msg.YoutubeID] = true
		m.syncDownloadStateToLibrary()
		youtubeID, title, artist := msg.YoutubeID, msg.Title, msg.Artist
		database := m.database
		return m, func() tea.Msg {
			result := downloadCmd(youtubeID, title, artist)()
			fp, _ := result.(string)
			if fp != "" {
				_ = database.MarkDownloaded(youtubeID, fp)
			}
			return screens.DownloadFinishedMsg{YoutubeID: youtubeID}
		}
	case screens.DownloadFinishedMsg:
		delete(m.downloading, msg.YoutubeID)
		m.downloaded[msg.YoutubeID] = true
		m.syncDownloadStateToLibrary()
		return m, nil
	case quitMsg:
		player.Stop()
		m.logs.Info("Command executed: Quit")
		return m, tea.Quit
	case commandsExecutedMsg:
		m.logs.Info(fmt.Sprintf("Command executed: %s", msg.Title))
		return m, msg.Cmd
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	if m.showCommandPalette {
		palette, cmd := m.commandPalette.Update(msg)
		m.commandPalette = palette
		return m, cmd
	}

	active := m.screens[m.activeScreen]
	updated, cmd := active.Update(msg)
	m.screens[m.activeScreen] = updated
	return m, cmd
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.ForceQuit) {
		player.Stop()
		return m, tea.Quit
	}

	if m.showCommandPalette {
		palette, cmd := m.commandPalette.Update(msg)
		m.commandPalette = palette
		if action := m.commandPalette.Action(); action.Type != commands.PaletteActionNone {
			return m.handlePaletteAction(action)
		}
		if executed := m.commandPalette.ExecutedCommand(); executed != nil {
			m.showCommandPalette = false
			m.commandPalette.Reset(m.theme.Name, m.paletteContext())
			return m, func() tea.Msg { return commandsExecutedMsg{Title: executed.Title, Cmd: executed.Run()} }
		}
		return m, cmd
	}

	if m.showHelp {
		if key.Matches(msg, m.keys.Cancel) || key.Matches(msg, m.keys.Help) {
			m.showHelp = false
		}
		return m, nil
	}

	// If the active screen wants to capture this key (e.g. search input active),
	// forward directly before any global handler runs.
	active := m.screens[m.activeScreen]
	if capturer, ok := active.(tuimod.KeyCapturer); ok && capturer.CapturesKey(msg) {
		updated, cmd := active.Update(msg)
		m.screens[m.activeScreen] = updated
		return m, cmd
	}

	switch {
	case key.Matches(msg, m.keys.Commands):
		m.showCommandPalette = true
		m.commandPalette.Reset(m.theme.Name, m.paletteContext())
		return m, nil
	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
		return m, nil
	case key.Matches(msg, m.keys.Cancel):
		return m, nil
	case key.Matches(msg, m.keys.Focus):
		if m.focus == FocusMain && m.showSidebar {
			m.focus = FocusSidebar
		} else {
			m.focus = FocusMain
		}
		return m, nil
	case key.Matches(msg, m.keys.Quit):
		player.Stop()
		return m, tea.Quit
	}

	if m.focus == FocusSidebar && m.showSidebar {
		return m.handleSidebarKey(msg)
	}

	updated, cmd := active.Update(msg)
	m.screens[m.activeScreen] = updated
	return m, cmd
}

func (m Model) handlePaletteAction(action commands.PaletteAction) (tea.Model, tea.Cmd) {
	m.commandPalette.ClearAction()
	switch action.Type {
	case commands.PaletteActionClose:
		m.showCommandPalette = false
		m.commandPalette.Reset(m.theme.Name, m.paletteContext())
		return m, nil
	case commands.PaletteActionExecute:
		m.showCommandPalette = false
		m.commandPalette.Reset(m.theme.Name, m.paletteContext())
		return m, func() tea.Msg { return commandsExecutedMsg{Title: action.Command.Title, Cmd: action.Command.Run()} }
	case commands.PaletteActionPreviewTheme:
		m.theme = *action.Theme
		m.updateDerivedScreens()
		return m, nil
	case commands.PaletteActionConfirmTheme:
		m.theme = *action.Theme
		m.logs.Info(fmt.Sprintf("Theme selected: %s", m.theme.Name))
		m.updateDerivedScreens()
		m.showCommandPalette = false
		m.commandPalette.Reset(m.theme.Name, m.paletteContext())
		return m, nil
	case commands.PaletteActionCancelTheme:
		m.theme = *action.Theme
		m.updateDerivedScreens()
		return m, nil
	}
	return m, nil
}

func (m Model) paletteContext() commands.Context {
	return commands.Context{ActiveScreen: m.activeScreen}
}

func (m Model) handleSidebarKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	idx := 0
	for i, id := range m.screenOrder {
		if id == m.activeScreen {
			idx = i
			break
		}
	}
	if key.Matches(msg, m.keys.Up) && idx > 0 {
		idx--
	} else if key.Matches(msg, m.keys.Down) && idx < len(m.screenOrder)-1 {
		idx++
	} else if !key.Matches(msg, m.keys.Enter) {
		return m, nil
	}
	m.switchScreen(m.screenOrder[idx])
	m.updateDerivedScreens()
	return m, nil
}

var downloadPath = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "Music/tuitube"
	}
	return home + "/Music/tuitube"
}()

func nowPlayingTick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return nowPlayingTickMsg{} })
}

// sessionResumeCmd restarts playback if the state file has a track but mpv isn't running.
func sessionResumeCmd() tea.Cmd {
	return func() tea.Msg {
		if player.IsAlive() {
			return nil // mpv already running, nothing to do
		}
		s := player.NowPlaying()
		if s == nil || s.Finished {
			return nil
		}
		// state file has a track but no live mpv — re-play it
		_ = player.Play(s.YoutubeID, s.Title, s.Artist)
		return nowPlayingTickMsg{}
	}
}

func downloadCmd(youtubeID, title, artist string) tea.Cmd {
	return func() tea.Msg {
		_ = os.MkdirAll(downloadPath, 0755)
		filename := sanitizeFilename(artist, title)
		filepath := downloadPath + "/" + filename + ".mp3"
		url := "https://www.youtube.com/watch?v=" + youtubeID
		cmd := exec.Command("yt-dlp",
			"-x", "--audio-format", "mp3",
			"-o", filepath,
			url,
		)
		_ = cmd.Run() // block until done so caller gets the finish signal
		return filepath
	}
}

// sanitizeFilename builds a clean "Artist - Title" filename from our DB data,
// stripping characters that are unsafe on common filesystems.
func sanitizeFilename(artist, title string) string {
	unsafe := `/\:*?"<>|`
	clean := func(s string) string {
		out := make([]rune, 0, len(s))
		for _, r := range s {
			if strings.ContainsRune(unsafe, r) {
				out = append(out, '-')
			} else {
				out = append(out, r)
			}
		}
		return strings.TrimSpace(string(out))
	}
	if artist == "" {
		return clean(title)
	}
	return clean(artist) + " - " + clean(title)
}

// buildQueue snapshots the library track list starting from the playing track.
func (m *Model) buildQueue(youtubeID string) {
	lib, ok := m.screens["library"].(screens.Library)
	if !ok {
		return
	}
	tracks := lib.Tracks()
	m.queue = tracks
	m.queuePos = 0
	for i, t := range tracks {
		if t.YoutubeID == youtubeID {
			m.queuePos = i
			break
		}
	}
}

// playNextInQueue advances the queue and plays the next track, or stops if done.
func (m *Model) playNextInQueue() tea.Cmd {
	next := m.queuePos + 1
	if next >= len(m.queue) {
		player.Stop()
		m.nowPlaying = nil
		m.syncNowPlayingToLibrary()
		return nowPlayingTick()
	}
	m.queuePos = next
	t := m.queue[next]
	if err := player.Play(t.YoutubeID, t.SongTitle, t.Artist); err != nil {
		m.logs.Error("autoplay", err)
	}
	m.nowPlaying = player.NowPlaying()
	m.syncNowPlayingToLibrary()
	return nowPlayingTick()
}

func (m *Model) syncDownloadStateToLibrary() {
	if lib, ok := m.screens["library"].(screens.Library); ok {
		m.screens["library"] = lib.WithDownloadState(m.downloading, m.downloaded)
	}
}

func (m *Model) syncNowPlayingToLibrary() {
	id, paused := "", false
	if m.nowPlaying != nil {
		id = m.nowPlaying.YoutubeID
		paused = m.nowPlaying.Paused
	}
	if lib, ok := m.screens["library"].(screens.Library); ok {
		m.screens["library"] = lib.WithNowPlaying(id, paused)
	}
}

func (m *Model) updateDerivedScreens() {
	lib := m.screens["library"]
	if l, ok := lib.(screens.Library); ok {
		m.screens["library"] = l.WithTheme(m.theme)
	}
	pl := m.screens["playlists"]
	if p, ok := pl.(screens.Playlists); ok {
		m.screens["playlists"] = p.WithTheme(m.theme)
	}
	m.screens["settings"] = screens.NewSettings(screens.SettingsState{
		ThemeName:      m.theme.Name,
		SidebarVisible: m.showSidebar,
		Version:        m.meta.Version,
		Commit:         m.meta.Commit,
		Date:           m.meta.Date,
	}, m.theme)
	m.screens["help"] = screens.NewHelp(m.keys.FullHelp(), m.theme)
	m.screens["logs"] = screens.NewLogs(m.logs)
}
