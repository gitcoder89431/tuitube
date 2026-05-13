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
	offset   int
}

func NewHelp(bindings [][]key.Binding, t theme.Theme) Help {
	return Help{bindings: bindings, theme: t}
}

func (h Help) WithTheme(t theme.Theme) Help {
	h.theme = t
	return h
}

func (h Help) Init() tea.Cmd { return nil }

func (h Help) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "up", "k":
			if h.offset > 0 {
				h.offset--
			}
		case "down", "j":
			h.offset++
		}
	}
	return h, nil
}

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
		"",
		row("enter", "play / pause"),
		row("n", "next track"),
		row("p", "pause / resume"),
		row("space", "toggle favorite"),
		row("f", "filter favorites"),
		row("d", "download to ~/Music/tuitube"),
		row("/", "search  (esc to clear)"),
		"",
		rule("Playlists"),
		"",
		row("enter", "open playlist"),
		row("n", "new playlist"),
		row("esc", "back to playlists"),
		"",
		rule("Playback"),
		"",
		row("← →", "seek -5s / +5s"),
		row("v", "visualizer — matrix / synthwave"),
		"",
		rule("Global"),
		"",
		row("ctrl+k", "command palette"),
		row("ctrl+t", "cycle theme"),
		row("tab", "focus sidebar / main"),
		row("?", "toggle this help"),
		row("q", "quit"),
	}

	if h.offset > max(0, len(lines)-height) {
		h.offset = max(0, len(lines)-height)
	}
	end := min(h.offset+height, len(lines))
	return lipgloss.NewStyle().Width(width).Height(height).Render(
		strings.Join(lines[h.offset:end], "\n"),
	)
}

func (h Help) Title() string { return "Help" }

func (h Help) KeyBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "scroll")),
		key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "scroll")),
	}
}
