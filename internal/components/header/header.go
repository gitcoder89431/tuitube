package header

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/gitcoder89431/tuitube/internal/player"
	"github.com/gitcoder89431/tuitube/internal/theme"
)

type Model struct {
	AppName     string
	ScreenTitle string
	Version     string
	NowPlaying  *player.State
	TimePos     float64
	Duration    float64
}

func fmtTime(secs float64) string {
	s := int(secs)
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

func View(m Model, width, height int, t theme.Theme) string {
	frameWidth, _ := t.Header.GetFrameSize()
	innerWidth := max(0, width-frameWidth)

	left := t.Title.Render(m.AppName)

	right := m.ScreenTitle
	rightW := lipgloss.Width(right)

	const barW = 16

	if m.NowPlaying != nil {
		icon := "▶"
		if m.NowPlaying.Paused {
			icon = "⏸"
		}
		label := icon + "  " + m.NowPlaying.Title
		if m.NowPlaying.Artist != "" {
			label = icon + "  " + m.NowPlaying.Artist + " — " + m.NowPlaying.Title
		}

		progressW := 0
		var progressRender string
		if m.Duration > 0 {
			tpStr := fmtTime(m.TimePos)
			durStr := fmtTime(m.Duration)
			filled := max(0, min(barW, int(m.TimePos/m.Duration*float64(barW))))
			bar := t.Success.Render(strings.Repeat("-", filled)) + t.Muted.Render(strings.Repeat("-", barW-filled))
			progressW = lipgloss.Width("  |  " + tpStr + " " + strings.Repeat("-", barW) + " " + durStr)
			progressRender = t.Muted.Render("  |  ") + t.Muted.Render(tpStr) + " " + bar + " " + t.Muted.Render(durStr)
		}

		maxLabelW := innerWidth - rightW - progressW - 4
		if maxLabelW > 8 {
			runes := []rune(label)
			if len(runes) > maxLabelW {
				label = string(runes[:maxLabelW-1]) + "…"
			}
			track := t.Accent.Render(label) + progressRender
			gap := max(0, innerWidth-lipgloss.Width(track)-rightW)
			content := track + lipgloss.NewStyle().Width(gap).Render("") + t.Muted.Render(right)
			return t.Header.Width(width).Height(height).Render(content)
		}
	}

	gap := max(0, innerWidth-lipgloss.Width(left)-rightW)
	content := left + lipgloss.NewStyle().Width(gap).Render("") + t.Muted.Render(right)
	return t.Header.Width(width).Height(height).Render(content)
}
