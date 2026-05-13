package screens

import (
	"fmt"
	"strings"

	"github.com/gitcoder89431/tui-tube/internal/agentlog"
	"github.com/gitcoder89431/tui-tube/internal/debug"
	"github.com/gitcoder89431/tui-tube/internal/theme"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
)

type logLine struct {
	text    string
	isAgent bool
}

type Logs struct {
	log    *debug.Log
	theme  theme.Theme
	offset int
}

func NewLogs(log *debug.Log, t theme.Theme) Logs {
	return Logs{log: log, theme: t}
}

func (l Logs) WithTheme(t theme.Theme) Logs {
	l.theme = t
	return l
}

func (l Logs) Init() tea.Cmd { return nil }

func (l Logs) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case "up", "k":
			if l.offset > 0 {
				l.offset--
			}
		case "down", "j":
			lines := l.buildLines()
			if l.offset < max(0, len(lines)-1) {
				l.offset++
			}
		}
	}
	return l, nil
}

func (l Logs) View(width, height int) string {
	lines := l.buildLines()
	if len(lines) == 0 {
		return lipgloss.NewStyle().Width(width).Height(height).Render(
			l.theme.Muted.Render("No activity yet."),
		)
	}
	if l.offset > max(0, len(lines)-1) {
		l.offset = 0
	}
	end := min(len(lines), l.offset+height)
	var rendered []string
	for _, line := range lines[l.offset:end] {
		if line.isAgent {
			rendered = append(rendered, l.theme.Accent.Render(line.text))
		} else {
			rendered = append(rendered, l.theme.Muted.Render(line.text))
		}
	}
	return lipgloss.NewStyle().Width(width).Height(height).Render(strings.Join(rendered, "\n"))
}

func (l Logs) buildLines() []logLine {
	var lines []logLine

	// agent log — newest first
	for _, e := range agentlog.Read() {
		lines = append(lines, logLine{
			text:    fmt.Sprintf("%s  agent  %s", e.Time.Format("15:04:05"), e.Message),
			isAgent: true,
		})
	}

	// app log — reverse so newest is near top
	appEntries := l.log.Entries()
	for i := len(appEntries) - 1; i >= 0; i-- {
		e := appEntries[i]
		lines = append(lines, logLine{
			text:    fmt.Sprintf("%s  %-5s  %s", e.Time.Format("15:04:05"), e.Level, e.Message),
			isAgent: false,
		})
	}

	return lines
}

func (l Logs) Title() string { return "Logs" }

func (l Logs) KeyBindings() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "scroll up")),
		key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "scroll down")),
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
