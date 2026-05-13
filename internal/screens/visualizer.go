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
