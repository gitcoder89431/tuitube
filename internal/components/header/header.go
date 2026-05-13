package header

import (
	"fmt"

	"github.com/gitcoder89431/tui-tube/internal/player"
	"github.com/gitcoder89431/tui-tube/internal/theme"
	"charm.land/lipgloss/v2"
)

type Model struct {
	AppName     string
	ScreenTitle string
	Version     string
	NowPlaying  *player.State
	TimePos     float64
	Duration    float64
}

func View(m Model, width, height int, t theme.Theme) string {
	frameWidth, _ := t.Header.GetFrameSize()
	innerWidth := max(0, width-frameWidth)

	left := t.Title.Render(m.AppName)

	right := m.ScreenTitle
	rightW := lipgloss.Width(right)

	center := ""
	if m.NowPlaying != nil {
		icon := "▶"
		if m.NowPlaying.Paused {
			icon = "⏸"
		}
		label := icon + "  " + m.NowPlaying.Title
		if m.NowPlaying.Artist != "" {
			label = icon + "  " + m.NowPlaying.Artist + " — " + m.NowPlaying.Title
		}
		progress := ""
		if m.Duration > 0 {
			progress = "  |  " + fmtTime(m.TimePos) + "/" + fmtTime(m.Duration)
		}
		sep := t.Muted.Render("  |  ")
		sepW := lipgloss.Width(sep)
		leftW := lipgloss.Width(left)
		progressW := lipgloss.Width(progress)
		maxLabelW := innerWidth - leftW - sepW - rightW - progressW - 4
		if maxLabelW > 8 {
			runes := []rune(label)
			if len(runes) > maxLabelW {
				label = string(runes[:maxLabelW-1]) + "…"
			}
			center = sep + t.Accent.Render(label) + t.Muted.Render(progress)
		}
	}

	gap := max(0, innerWidth-lipgloss.Width(left)-lipgloss.Width(center)-rightW)
	content := left + center + lipgloss.NewStyle().Width(gap).Render("") + t.Muted.Render(right)

	return t.Header.Width(width).Height(height).Render(content)
}

func fmtTime(secs float64) string {
	s := int(secs)
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
