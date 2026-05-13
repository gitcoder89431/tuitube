package screens

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/gitcoder89431/tui-tube/internal/theme"
)

type Help struct {
	bindings [][]key.Binding
	theme    theme.Theme
}

func NewHelp(bindings [][]key.Binding, t theme.Theme) Help {
	return Help{bindings: bindings, theme: t}
}

func (h Help) WithTheme(t theme.Theme) Help {
	h.theme = t
	return h
}

func (h Help) Init() tea.Cmd                             { return nil }
func (h Help) Update(msg tea.Msg) (Screen, tea.Cmd)     { return h, nil }
func (h Help) Title() string                             { return "Help" }
func (h Help) KeyBindings() []key.Binding                { return nil }

func (h Help) View(width, height int) string {
	rule := func(label string) string {
		labelW := lipgloss.Width(label)
		fill := strings.Repeat("/", max(0, width-labelW-1))
		return h.theme.Title.Render(label) + h.theme.PaletteAccent.Render(" "+fill)
	}
	row := func(keys, desc string) string {
		return "  " + h.theme.Accent.Render(keys) + "  " + h.theme.Text.Render(desc)
	}

	lines := []string{
		rule("Library"),
		row("enter", "play / pause current track"),
		row("n  p  space", "next  |  pause  |  favorite"),
		row("f", "toggle favorites filter"),
		row("d", "download to ~/Music/tuitube"),
		row("/", "search — esc to clear"),
		"",
		rule("Playlists"),
		row("enter", "open playlist"),
		row("n  esc", "new playlist  |  back to playlists"),
		"",
		rule("Playback"),
		row("← →", "seek -5s / +5s"),
		row("v", "visualizer — matrix / synthwave"),
		"",
		rule("Global"),
		row("ctrl+k", "command palette"),
		row("ctrl+t", "cycle theme"),
		row("tab", "focus sidebar / main"),
		row("q", "quit"),
	}

	return lipgloss.NewStyle().Width(width).Height(height).Render(
		strings.Join(lines, "\n"),
	)
}
