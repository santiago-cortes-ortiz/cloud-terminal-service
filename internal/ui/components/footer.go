package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"aws-terminal/internal/ui/styles"
)

type FooterProps struct {
	Width       int
	Help        string
	StatusParts []string
}

func RenderFooter(props FooterProps) string {
	if props.Width <= 0 {
		return ""
	}

	lines := []string{props.Help}
	if len(props.StatusParts) > 0 {
		lines = append(lines, styles.MutedStyle.Render(strings.Join(props.StatusParts, " • ")))
	}

	return lipgloss.NewStyle().
		Width(props.Width).
		MarginTop(1).
		Render(strings.Join(lines, "\n"))
}
