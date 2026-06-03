package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/gitcoder89431/tuitube/internal/commands"
	"github.com/gitcoder89431/tuitube/internal/player"
	"github.com/gitcoder89431/tuitube/internal/screens"
	"github.com/gitcoder89431/tuitube/internal/theme"
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
	case visTickMsg:
		if m.visMode != screens.VisModeOff {
			m.visFrame++
			return m, visTick()
		}
		return m, nil
	case nowPlayingTickMsg:
		state := m.player.NowPlaying()
		if state != nil && state.Finished {
			return m, m.playNextInQueue()
		}
		m.nowPlaying = state
		if state != nil && state.Playing && !state.Paused {
			m.timePos, m.duration = m.player.Progress()
		}
		m.syncNowPlayingToLibrary()
		return m, nowPlayingTick()
	case screens.BackMsg:
		m.switchScreen("playlists")
		if lib, ok := m.screens["library"].(screens.Library); ok {
			m.screens["library"] = lib.WithPlaylistTitle("")
			return m, lib.ReloadCmd()
		}
		return m, nil
	case screens.TracksLoadedMsg:
		// always route to library regardless of active screen (e.g. after esc back to playlists)
		if lib, ok := m.screens["library"].(screens.Library); ok {
			updated, cmd := lib.Update(msg)
			m.screens["library"] = updated
			return m, cmd
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
		if err := m.player.TogglePause(); err != nil {
			m.logs.Error("toggle pause", err)
		}
		m.nowPlaying = m.player.NowPlaying()
		return m, nil
	case screens.PlayTrackMsg:
		// same track → toggle pause instead of restarting
		if m.nowPlaying != nil && m.nowPlaying.YoutubeID == msg.YoutubeID {
			if err := m.player.TogglePause(); err != nil {
				m.logs.Error("toggle pause", err)
			}
		} else {
			if err := m.player.Play(msg.YoutubeID, msg.Title, msg.Artist, m.downloaded[msg.YoutubeID], 0); err != nil {
				m.logs.Error("mpv", err)
			}
			m.buildQueue(msg.YoutubeID)
		}
		m.nowPlaying = m.player.NowPlaying()
		m.syncNowPlayingToLibrary()
		return m, nil
	case screens.DownloadTrackMsg:
		m.downloading[msg.YoutubeID] = true
		m.syncDownloadStateToLibrary()
		return m, downloadCmd(msg.YoutubeID, msg.Title, msg.Artist)
	case downloadResult:
		if msg.err != nil {
			m.logs.Error("download", msg.err)
		} else if m.database != nil {
			_ = m.database.MarkDownloaded(msg.youtubeID, msg.filepath)
		}
		delete(m.downloading, msg.youtubeID)
		if msg.filepath != "" {
			m.downloaded[msg.youtubeID] = msg.filepath
		}
		m.syncDownloadStateToLibrary()
		return m, func() tea.Msg {
			return screens.DownloadFinishedMsg{YoutubeID: msg.youtubeID, Filepath: msg.filepath}
		}
	case screens.DownloadFinishedMsg:
		// state already updated in downloadResult; nothing to do at app level
		return m, nil
	case quitMsg:
		m.player.SavePosition()
		m.player.Stop()
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
		m.player.Stop()
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
	if capturer, ok := active.(screens.KeyCapturer); ok && capturer.CapturesKey(msg) {
		updated, cmd := active.Update(msg)
		m.screens[m.activeScreen] = updated
		return m, cmd
	}

	switch {
	case key.Matches(msg, m.keys.Sidebar):
		m.showSidebar = !m.showSidebar
		m.updateDerivedScreens()
		return m, nil
	case key.Matches(msg, m.keys.Next):
		return m, m.playNextInQueue()
	case key.Matches(msg, m.keys.SeekForward):
		_ = m.player.SeekForward()
		return m, nil
	case key.Matches(msg, m.keys.SeekBackward):
		_ = m.player.SeekBackward()
		return m, nil
	case key.Matches(msg, m.keys.CycleTheme):
		themes := theme.BuiltIns()
		for i, t := range themes {
			if t.Name == m.theme.Name {
				m.theme = themes[(i+1)%len(themes)]
				break
			}
		}
		m.updateDerivedScreens()
		return m, nil
	case key.Matches(msg, m.keys.Visualizer):
		m.visMode = m.visMode.Next()
		if m.visMode != screens.VisModeOff {
			return m, visTick()
		}
		return m, nil
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
		m.player.Stop()
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

func resolveDownloadPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, "Music", "tuitube"), nil
}

func nowPlayingTick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return nowPlayingTickMsg{} })
}

func visTick() tea.Cmd {
	return tea.Tick(33*time.Millisecond, func(time.Time) tea.Msg { return visTickMsg{} })
}

// sessionResumeCmd restarts playback if the state file has a track but mpv isn't running.
func sessionResumeCmd(p *player.Player, downloaded map[string]string) tea.Cmd {
	return func() tea.Msg {
		if p.IsAlive() {
			return nil // mpv already running, nothing to do
		}
		s := p.NowPlaying()
		if s == nil || s.Finished {
			return nil
		}
		_ = p.Play(s.YoutubeID, s.Title, s.Artist, downloaded[s.YoutubeID], s.ResumePos)
		return nowPlayingTickMsg{}
	}
}

type downloadResult struct {
	youtubeID string
	filepath  string
	err       error
}

func downloadCmd(youtubeID, title, artist string) tea.Cmd {
	return func() tea.Msg {
		dlPath, err := resolveDownloadPath()
		if err != nil {
			return downloadResult{youtubeID: youtubeID, err: err}
		}
		_ = os.MkdirAll(dlPath, 0755)
		fp := filepath.Join(dlPath, sanitizeFilename(artist, title)+".mp3")
		url := "https://www.youtube.com/watch?v=" + youtubeID
		cmd := exec.Command("yt-dlp",
			"-x", "--audio-format", "mp3",
			"-o", fp,
			url,
		)
		if err := cmd.Run(); err != nil {
			return downloadResult{youtubeID: youtubeID, err: err}
		}
		return downloadResult{youtubeID: youtubeID, filepath: fp}
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
		m.player.Stop()
		m.nowPlaying = nil
		m.syncNowPlayingToLibrary()
		return nowPlayingTick()
	}
	m.queuePos = next
	t := m.queue[next]
	if err := m.player.Play(t.YoutubeID, t.SongTitle, t.Artist, m.downloaded[t.YoutubeID], 0); err != nil {
		m.logs.Error("autoplay", err)
	}
	m.nowPlaying = m.player.NowPlaying()
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
	m.screens["logs"] = screens.NewLogs(m.logs, m.theme)
}
