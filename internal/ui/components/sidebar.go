package components

import (
	"strings"

	"aws-terminal/internal/ui/styles"
)

type SidebarItemState int

const (
	SidebarItemDefault SidebarItemState = iota
	SidebarItemSelected
	SidebarItemFocusedSelected
	SidebarItemActive
	SidebarItemFocusedActive
)

type SidebarItem struct {
	Label string
	State SidebarItemState
}

type SidebarSection struct {
	Title    string
	Focused  bool
	Items    []SidebarItem
	Content  string
	MaxLines int
}

type SidebarProps struct {
	Width    int
	Height   int
	Sections []SidebarSection
	Hint     string
}

func RenderSidebar(props SidebarProps) string {
	if props.Width <= 0 || props.Height <= 0 {
		return ""
	}

	lines := make([]string, 0, len(props.Sections)*4)
	for sectionIndex, section := range props.Sections {
		if sectionIndex > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, renderSectionTitle(section.Title, section.Focused))
		if section.Content != "" {
			lines = append(lines, constrainLines(section.Content, section.MaxLines))
			continue
		}
		for _, item := range section.Items {
			lines = append(lines, renderItem(item))
		}
	}

	if props.Hint != "" {
		lines = append(lines, "", styles.MutedStyle.Render(props.Hint))
	}

	return styles.RenderBox(styles.SidebarPanelStyle, props.Width, props.Height, strings.Join(lines, "\n"))
}

func renderSectionTitle(title string, focused bool) string {
	if focused {
		return styles.FocusedSectionTitleStyle.Render(title)
	}

	return styles.SidebarTitleStyle.Render(title)
}

func renderItem(item SidebarItem) string {
	switch item.State {
	case SidebarItemFocusedActive:
		return styles.FocusedActiveProfileStyle.Render(item.Label)
	case SidebarItemActive:
		return styles.ActiveProfileStyle.Render(item.Label)
	case SidebarItemFocusedSelected:
		return styles.FocusedSelectedSidebarItemStyle.Render(item.Label)
	case SidebarItemSelected:
		return styles.SelectedSidebarItemStyle.Render(item.Label)
	default:
		return styles.SidebarItemStyle.Render(item.Label)
	}
}

func constrainLines(content string, maxLines int) string {
	if maxLines <= 0 || content == "" {
		return content
	}

	lines := strings.Split(content, "\n")
	if len(lines) <= maxLines {
		return content
	}

	return strings.Join(lines[:maxLines], "\n")
}
