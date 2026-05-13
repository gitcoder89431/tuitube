package screens

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/gitcoder89431/tui-tube/internal/theme"
)

// Half-width katakana + digits — the iconic Matrix digital rain characters.
var matrixChars = []rune{
	'ｦ', 'ｧ', 'ｨ', 'ｩ', 'ｪ', 'ｫ', 'ｬ', 'ｭ', 'ｮ', 'ｯ',
	'ｰ', 'ｱ', 'ｲ', 'ｳ', 'ｴ', 'ｵ', 'ｶ', 'ｷ', 'ｸ', 'ｹ', 'ｺ',
	'ｻ', 'ｼ', 'ｽ', 'ｾ', 'ｿ', 'ﾀ', 'ﾁ', 'ﾂ', 'ﾃ', 'ﾄ',
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
}

// VisMode selects the active visualizer.
type VisMode int

const (
	VisModeOff    VisMode = iota
	VisModeMatrix         // katakana / digit rain
	VisModeBinary         // scrolling 0s and 1s
	VisModeRain           // falling vertical drop streaks
	visModeCount
)

func (m VisMode) Next() VisMode { return (m + 1) % visModeCount }

// RenderMatrix renders one frame of matrix digital rain.
// frame is a monotonically increasing tick counter.
// Colors are pulled from the theme so it adapts to any palette.
func RenderMatrix(width, height int, frame uint64, t theme.Theme) string {
	// extract fg colors once — avoids per-cell lipgloss.Style.Render overhead
	headFg := t.Text.GetForeground()    // bright leading character
	midFg := t.Accent.GetForeground()  // near trail
	tailFg := t.Muted.GetForeground()  // fading tail

	cell := func(ch rune, fg color.Color) string {
		return lipgloss.NewStyle().Foreground(fg).Render(string(ch))
	}

	lines := make([]string, height)
	for row := 0; row < height; row++ {
		var sb strings.Builder
		for col := 0; col < width; col++ {
			seed := uint64(col)*7919 + 104729
			speed := 2 + int(seed%3)
			trailLen := 3 + int((seed/7)%3)
			cycleLen := height + trailLen + 6
			offset := int((seed / 13) % uint64(cycleLen))
			pos := (int(frame)/speed + offset) % cycleLen
			dist := pos - row

			if dist < 0 || dist > trailLen {
				sb.WriteByte(' ')
			} else {
				charSeed := seed ^ (uint64(row)*31 + (frame/4)*17)
				ch := matrixChars[charSeed%uint64(len(matrixChars))]
				switch {
				case dist == 0:
					sb.WriteString(cell(ch, headFg))
				case dist <= 2:
					sb.WriteString(cell(ch, midFg))
				default:
					sb.WriteString(cell(ch, tailFg))
				}
			}
		}
		lines[row] = sb.String()
	}
	return strings.Join(lines, "\n")
}

// RenderRain renders falling vertical drop streaks using box-drawing characters.
// Each column has independent speed and drop length for an organic feel.
func RenderRain(width, height int, frame uint64, t theme.Theme) string {
	headFg := t.Accent.GetForeground() // bright drop head
	midFg := t.Text.GetForeground()   // drop body
	tailFg := t.Muted.GetForeground() // fading tail

	cell := func(ch rune, fg color.Color) string {
		return lipgloss.NewStyle().Foreground(fg).Render(string(ch))
	}

	lines := make([]string, height)
	for row := 0; row < height; row++ {
		var sb strings.Builder
		for col := 0; col < width; col++ {
			seed := uint64(col)*7919 + 104729
			speed := 1 + int(seed%3)         // 1-3 frames per step
			dropLen := 2 + int((seed/7)%3)   // 2-4 chars tall
			cycleLen := height + dropLen + 3
			offset := int((seed / 13) % uint64(cycleLen))
			pos := (int(frame)/speed + offset) % cycleLen
			dist := pos - row

			if dist >= 0 && dist < dropLen {
				switch {
				case dist == 0:
					sb.WriteString(cell('┃', headFg))
				case dist == 1:
					sb.WriteString(cell('│', midFg))
				default:
					sb.WriteString(cell(':', tailFg))
				}
			} else {
				sb.WriteByte(' ')
			}
		}
		lines[row] = sb.String()
	}
	return strings.Join(lines, "\n")
}

// RenderBinary renders scrolling columns of 0s and 1s.
// Columns scroll at different speeds; 1s are brighter than 0s.
func RenderBinary(width, height int, frame uint64, t theme.Theme) string {
	brightFg := t.Accent.GetForeground() // bright 1s
	midFg := t.Text.GetForeground()     // normal 1s
	dimFg := t.Muted.GetForeground()    // 0s

	cell := func(ch byte, fg color.Color) string {
		return lipgloss.NewStyle().Foreground(fg).Render(string(ch))
	}

	lines := make([]string, height)
	for row := 0; row < height; row++ {
		var sb strings.Builder
		for col := 0; col < width; col++ {
			seed := uint64(col)*7919 + uint64(row)*6271
			// each column scrolls at a distinct speed, scaled down for readability
			speed := uint64(3 + seed%5)
			scroll := int(frame / speed)

			// deterministic bit: hash of (col, scrolled-row)
			h := seed ^ (uint64(scroll+row)*104729)
			h ^= h >> 16
			h *= 0x45d9f3b37197344b
			h ^= h >> 16
			prob := h % 100

			// ~35% ones, ~65% zeros
			if prob < 35 {
				// brighter 1 for the leading edge of each column's scroll
				if (scroll+row)%height < 3 {
					sb.WriteString(cell('1', brightFg))
				} else {
					sb.WriteString(cell('1', midFg))
				}
			} else {
				sb.WriteString(cell('0', dimFg))
			}
		}
		lines[row] = sb.String()
	}
	return strings.Join(lines, "\n")
}
