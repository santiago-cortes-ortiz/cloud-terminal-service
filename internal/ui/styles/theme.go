package styles

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	DocStyle = lipgloss.NewStyle().
			Padding(1, 2)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86"))

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	SectionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("81"))

	FocusedSectionTitleStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("230")).
					Background(lipgloss.Color("62")).
					Padding(0, 1)

	StatusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212"))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203"))

	MutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	SidebarPanelStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62")).
				Padding(1, 1)

	PanelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2)

	SidebarTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("111"))

	SidebarItemStyle = lipgloss.NewStyle().
				PaddingLeft(1)

	SelectedSidebarItemStyle = lipgloss.NewStyle().
					PaddingLeft(1).
					Bold(true).
					Foreground(lipgloss.Color("111"))

	FocusedSelectedSidebarItemStyle = lipgloss.NewStyle().
					PaddingLeft(1).
					Bold(true).
					Foreground(lipgloss.Color("230")).
					Background(lipgloss.Color("63"))

	ActiveProfileStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Foreground(lipgloss.Color("120"))

	FocusedActiveProfileStyle = lipgloss.NewStyle().
					PaddingLeft(1).
					Bold(true).
					Foreground(lipgloss.Color("22")).
					Background(lipgloss.Color("120"))
)

func RenderBox(style lipgloss.Style, width, height int, content string) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	return style.
		Width(width).
		Height(height).
		MaxWidth(width).
		MaxHeight(height).
		Render(content)
}
