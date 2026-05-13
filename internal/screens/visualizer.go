package screens

import (
	"image/color"
	"math"
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
	VisModeRetro          // synthwave sunset + perspective grid
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

// brailleBit maps [row][col] inside a 4×2 braille cell to its Unicode bit.
var brailleBit = [4][2]rune{
	{0x01, 0x08},
	{0x02, 0x10},
	{0x04, 0x20},
	{0x40, 0x80},
}

// RenderRetro draws a synthwave scene: striped setting sun, perspective grid
// floor scrolling toward the viewer, and a slow sine wave at the horizon.
// Uses braille characters for sub-cell resolution. Ported from cliamp (MIT).
func RenderRetro(width, height int, frame uint64, t theme.Theme) string {
	gridFg := t.Muted.GetForeground()   // perspective grid
	sunFg := t.Warn.GetForeground()     // sunset amber/orange
	waveFg := t.Accent.GetForeground()  // neon horizon wave

	dotRows := height * 4
	dotCols := width * 2

	horizonDot := max(dotRows*2/5, 2)
	floorRows := dotRows - horizonDot
	centerX := float64(dotCols-1) / 2.0

	grid := make([]byte, dotRows*dotCols) // 0=empty 1=grid 2=wave 3=sun

	// ── SUN: striped semicircle above horizon ──
	sunR := float64(horizonDot) * 0.85
	for dy := 0; dy < horizonDot; dy++ {
		rowDist := float64(horizonDot - dy)
		if rowDist > sunR {
			continue
		}
		halfW := math.Sqrt(sunR*sunR - rowDist*rowDist)
		if rowDist < sunR*0.5 {
			sw := max(1, int(sunR*0.15))
			if (int(rowDist)/sw)%2 == 1 {
				continue
			}
		}
		left := max(0, int(centerX-halfW))
		right := min(dotCols-1, int(centerX+halfW))
		for dx := left; dx <= right; dx++ {
			grid[dy*dotCols+dx] = 3
		}
	}

	// ── HORIZON LINE ──
	for dx := 0; dx < dotCols; dx++ {
		grid[horizonDot*dotCols+dx] = 1
	}

	// ── PERSPECTIVE GRID: vertical lines converging to vanishing point ──
	const numVLines = 18
	for i := 0; i <= numVLines; i++ {
		bottomX := float64(i) * float64(dotCols-1) / float64(numVLines)
		for dy := horizonDot + 1; dy < dotRows; dy++ {
			t2 := float64(dy-horizonDot) / float64(max(1, floorRows-1))
			screenX := centerX + (bottomX-centerX)*t2
			ix := int(math.Round(screenX))
			if ix >= 0 && ix < dotCols {
				grid[dy*dotCols+ix] = 1
			}
		}
	}

	// ── HORIZONTAL LINES scrolling toward viewer ──
	scroll := math.Mod(float64(frame)*0.08, 1.0)
	const numHLines = 10
	for i := 0; i < numHLines; i++ {
		z := (float64(i) + scroll) / float64(numHLines)
		if z > 1.0 {
			z -= 1.0
		}
		dy := horizonDot + 1 + int(z*z*float64(max(1, floorRows-2)))
		if dy > horizonDot && dy < dotRows {
			for dx := 0; dx < dotCols; dx++ {
				grid[dy*dotCols+dx] = 1
			}
		}
	}

	// ── SINE WAVE at horizon (slow, no audio needed) ──
	waveY := make([]int, dotCols)
	maxWave := float64(horizonDot) * 0.85
	for dx := 0; dx < dotCols; dx++ {
		phase := float64(dx)/float64(max(1, dotCols-1))*math.Pi*4 - float64(frame)*0.04
		level := 0.05 + 0.35*(1+math.Sin(phase))/2
		wy := horizonDot - int(level*maxWave)
		waveY[dx] = max(0, min(dotRows-1, wy))
	}
	for dx := 0; dx < dotCols; dx++ {
		y := waveY[dx]
		grid[y*dotCols+dx] = 2
		if dx > 0 {
			lo, hi := min(y, waveY[dx-1]), max(y, waveY[dx-1])
			for fy := lo; fy <= hi; fy++ {
				grid[fy*dotCols+dx] = 2
			}
		}
	}

	// ── RENDER braille cells ──
	lines := make([]string, height)
	for row := 0; row < height; row++ {
		var sb strings.Builder
		base := row * 4
		for ch := 0; ch < width; ch++ {
			var braille rune = '⠀'
			colBase := ch * 2
			hasWave, hasSun := false, false
			for dr := 0; dr < 4; dr++ {
				for dc := 0; dc < 2; dc++ {
					dy := base + dr
					dx := colBase + dc
					if dy >= dotRows || dx >= dotCols {
						continue
					}
					switch grid[dy*dotCols+dx] {
					case 1:
						braille |= brailleBit[dr][dc]
					case 2:
						braille |= brailleBit[dr][dc]
						hasWave = true
					case 3:
						braille |= brailleBit[dr][dc]
						hasSun = true
					}
				}
			}
			var fg color.Color
			switch {
			case hasWave:
				fg = waveFg
			case hasSun:
				fg = sunFg
			default:
				fg = gridFg
			}
			sb.WriteString(lipgloss.NewStyle().Foreground(fg).Render(string(braille)))
		}
		lines[row] = sb.String()
	}
	return strings.Join(lines, "\n")
}
