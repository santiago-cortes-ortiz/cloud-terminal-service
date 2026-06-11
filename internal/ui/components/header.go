package components

import (
	"github.com/charmbracelet/lipgloss"

	"aws-terminal/internal/ui/styles"
)

type HeaderProps struct {
	Width    int
	Title    string
	Subtitle string
	Status   string
}

func RenderHeader(props HeaderProps) string {
	if props.Width <= 0 {
		return ""
	}

	title := styles.TitleStyle.Render(props.Title)
	status := styles.StatusStyle.Render(props.Status)

	if props.Width < 52 {
		return lipgloss.NewStyle().Width(props.Width).Render(title)
	}

	row := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Width(max(0, props.Width-lipgloss.Width(status)-1)).Render(title),
		status,
	)

	if props.Width < 78 || props.Subtitle == "" {
		return lipgloss.NewStyle().Width(props.Width).Render(row)
	}

	subtitle := styles.SubtitleStyle.Render(props.Subtitle)
	return lipgloss.NewStyle().Width(props.Width).Render(lipgloss.JoinVertical(lipgloss.Left, row, subtitle))
}
