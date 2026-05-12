package app

import (
	"fmt"
	"os/exec"

	"github.com/gitcoder89431/tui-tube/internal/commands"
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
	case screens.PlayTrackMsg:
		m.stopPlayer()
		url := "https://www.youtube.com/watch?v=" + msg.YoutubeID
		cmd := exec.Command("mpv", "--no-video", "--really-quiet", url)
		if err := cmd.Start(); err != nil {
			m.logs.Error("mpv", err)
		} else {
			m.currentPlayer = cmd
		}
		return m, nil
	case screens.DownloadTrackMsg:
		url := "https://www.youtube.com/watch?v=" + msg.YoutubeID
		dl := exec.Command("yt-dlp",
			"-x", "--audio-format", "mp3",
			"-o", downloadPath+"/%(artist)s - %(title)s.%(ext)s",
			url,
		)
		if err := dl.Start(); err != nil {
			m.logs.Error("yt-dlp", err)
		}
		return m, nil
	case quitMsg:
		m.stopPlayer()
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
		m.stopPlayer()
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
		m.stopPlayer()
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

const downloadPath = "/music/yt-radio"

func (m *Model) stopPlayer() {
	if m.currentPlayer != nil && m.currentPlayer.Process != nil {
		_ = m.currentPlayer.Process.Kill()
		_ = m.currentPlayer.Wait()
		m.currentPlayer = nil
	}
}

func (m *Model) updateDerivedScreens() {
	lib := m.screens["library"]
	if l, ok := lib.(screens.Library); ok {
		m.screens["library"] = l.WithTheme(m.theme)
	}
	st := m.screens["stations"]
	if s, ok := st.(screens.Stations); ok {
		m.screens["stations"] = s.WithTheme(m.theme)
	}
	m.screens["settings"] = screens.NewSettings(screens.SettingsState{
		ThemeName:      m.theme.Name,
		SidebarVisible: m.showSidebar,
		Version:        m.meta.Version,
		Commit:         m.meta.Commit,
		Date:           m.meta.Date,
	})
	m.screens["help"] = screens.NewHelp(m.keys.FullHelp())
	m.screens["logs"] = screens.NewLogs(m.logs)
}
