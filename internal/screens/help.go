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

func (h Help) Init() tea.Cmd { return nil }

func (h Help) Update(msg tea.Msg) (Screen, tea.Cmd) { return h, nil }

func (h Help) View(width, height int) string {
	rule := func(label string) string {
		labelW := lipgloss.Width(label)
		fill := strings.Repeat("/", max(0, width-labelW-1))
		return h.theme.Title.Render(label) + h.theme.Muted.Render(" "+fill)
	}
	row := func(keys, desc string) string {
		return "  " + h.theme.Accent.Render(keys) + "  " + h.theme.Text.Render(desc) + "\n"
	}

	var b strings.Builder

	b.WriteString(rule("Library") + "\n\n")
	b.WriteString(row("enter", "play selected track (or pause if already playing)"))
	b.WriteString(row("p", "pause / resume"))
	b.WriteString(row("d", "download track to ~/Music/tuitube"))
	b.WriteString(row("space", "toggle favorite"))
	b.WriteString(row("f", "filter to favorites only"))
	b.WriteString(row("/", "search — live filter, esc to clear"))
	b.WriteString(row("j / k", "move down / up"))
	b.WriteString("\n")

	b.WriteString(rule("Playlists") + "\n\n")
	b.WriteString(row("enter", "open playlist"))
	b.WriteString(row("n", "create new playlist"))
	b.WriteString(row("esc", "back to playlists from library"))
	b.WriteString("\n")

	b.WriteString(rule("Global") + "\n\n")
	b.WriteString(row("ctrl+k", "command palette"))
	b.WriteString(row("ctrl+t", "cycle theme"))
	b.WriteString(row("tab", "focus sidebar / main"))
	b.WriteString(row("?", "toggle this help"))
	b.WriteString(row("q", "quit"))

	return lipgloss.NewStyle().Width(width).Height(height).Render(b.String())
}

func (h Help) Title() string { return "Help" }

func (h Help) KeyBindings() []key.Binding { return nil }
