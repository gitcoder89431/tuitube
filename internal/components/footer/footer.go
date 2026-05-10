package footer

import (
	"strings"

	"github.com/gitcoder89431/tui-tube/internal/theme"
	"github.com/charmbracelet/bubbles/key"
)

func View(bindings []key.Binding, width, height int, t theme.Theme) string {
	parts := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		help := binding.Help()
		parts = append(parts, help.Key+" "+help.Desc)
	}
	return t.Footer.Width(width).Height(height).Render(strings.Join(parts, "   "))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
