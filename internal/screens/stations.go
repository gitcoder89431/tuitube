package screens

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/bubbles/key"
	"github.com/gitcoder89431/tui-tube/internal/theme"
)

type Stations struct {
	theme theme.Theme
}

func NewStations(t theme.Theme) Stations {
	return Stations{theme: t}
}

func (s Stations) WithTheme(t theme.Theme) Stations {
	s.theme = t
	return s
}

func (s Stations) Init() tea.Cmd { return nil }

func (s Stations) Update(msg tea.Msg) (Screen, tea.Cmd) {
	return s, nil
}

func (s Stations) View(width, height int) string {
	return lipgloss.NewStyle().Width(width).Height(height).
		Render(s.theme.Muted.Render("Stations — coming soon"))
}

func (s Stations) Title() string { return "Stations" }

func (s Stations) KeyBindings() []key.Binding { return nil }
