package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type Theme struct {
	Name       string
	Background color.Color

	Text          lipgloss.Style
	Muted         lipgloss.Style
	Accent        lipgloss.Style
	Title         lipgloss.Style
	Selected      lipgloss.Style
	Disabled      lipgloss.Style
	Header        lipgloss.Style
	HeaderAccent  lipgloss.Style
	Sidebar       lipgloss.Style
	Main          lipgloss.Style
	Footer        lipgloss.Style
	Modal         lipgloss.Style
	PaletteAccent lipgloss.Style
	Border        lipgloss.Style
	Focus         lipgloss.Style
	Info          lipgloss.Style
	Success       lipgloss.Style
	Warn          lipgloss.Style
	Error         lipgloss.Style
}
