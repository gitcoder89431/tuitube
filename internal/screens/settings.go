package screens

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/gitcoder89431/tui-tube/internal/theme"
)

type SettingsState struct {
	ThemeName      string
	SidebarVisible bool
	Version        string
	Commit         string
	Date           string
}

type Settings struct {
	state SettingsState
	theme theme.Theme
}

func NewSettings(state SettingsState, t theme.Theme) Settings {
	return Settings{state: state, theme: t}
}

func (s Settings) WithTheme(t theme.Theme) Settings {
	s.theme = t
	return s
}

func (s Settings) Init() tea.Cmd { return nil }

func (s Settings) Update(msg tea.Msg) (Screen, tea.Cmd) { return s, nil }

func (s Settings) View(width, height int) string {
	rule := func(label string) string {
		labelW := lipgloss.Width(label)
		fill := strings.Repeat("/", max(0, width-labelW-1))
		return s.theme.Muted.Bold(true).Render(label) + s.theme.Muted.Render(" "+fill)
	}
	row := func(label, value string) string {
		return s.theme.Muted.Render(label+": ") + s.theme.Text.Render(value)
	}

	lines := []string{
		rule("Settings"),
		"",
		row("Theme", s.state.ThemeName),
		row("Sidebar", fmt.Sprintf("%t", s.state.SidebarVisible)),
		"",
		rule("Build"),
		"",
		row("Version", s.state.Version),
		row("Commit", s.state.Commit),
		row("Date", s.state.Date),
	}

	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

func (s Settings) Title() string { return "Settings" }

func (s Settings) KeyBindings() []key.Binding { return nil }
