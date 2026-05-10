package header

import (
	"fmt"

	"github.com/gitcoder89431/tui-tube/internal/theme"
	"charm.land/lipgloss/v2"
)

type Model struct {
	AppName     string
	ScreenTitle string
	Version     string
}

func View(m Model, width, height int, t theme.Theme) string {
	frameWidth, _ := t.Header.GetFrameSize()
	innerWidth := max(0, width-frameWidth)
	left := t.Title.Render(m.AppName)
	right := fmt.Sprintf("%s  %s", m.ScreenTitle, m.Version)
	content := lipgloss.JoinHorizontal(lipgloss.Center, left, lipgloss.PlaceHorizontal(max(0, innerWidth-lipgloss.Width(left)-lipgloss.Width(right)-2), lipgloss.Left, ""), right)
	return t.Header.Width(width).Height(height).Render(content)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
