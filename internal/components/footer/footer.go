package footer

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/gitcoder89431/tuitube/internal/theme"
)

func View(bindings []key.Binding, width, height int, t theme.Theme) string {
	parts := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		help := binding.Help()
		parts = append(parts, help.Key+" "+help.Desc)
	}
	return t.Footer.Width(width).Height(height).Render(strings.Join(parts, "   "))
}
