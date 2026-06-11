package shell

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"aws-terminal/internal/ui/styles"
)

type sidebarListItem interface {
	list.Item
	SidebarLabel() string
	SidebarActive() bool
}

type sidebarListDelegate struct {
	focused bool
}

func newSidebarListDelegate(focused bool) sidebarListDelegate {
	return sidebarListDelegate{focused: focused}
}

func (d sidebarListDelegate) Height() int {
	return 1
}

func (d sidebarListDelegate) Spacing() int {
	return 0
}

func (d sidebarListDelegate) Update(tea.Msg, *list.Model) tea.Cmd {
	return nil
}

func (d sidebarListDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	sidebarItem, ok := item.(sidebarListItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	prefix := "  "
	if isSelected {
		prefix = "▸ "
	}

	label := prefix + sidebarItem.SidebarLabel()
	style := styles.SidebarItemStyle
	switch {
	case isSelected && sidebarItem.SidebarActive() && d.focused:
		style = styles.FocusedActiveProfileStyle
	case isSelected && d.focused:
		style = styles.FocusedSelectedSidebarItemStyle
	case sidebarItem.SidebarActive():
		style = styles.ActiveProfileStyle
	case isSelected:
		style = styles.SelectedSidebarItemStyle
	}

	contentWidth := max(0, m.Width()-style.GetHorizontalFrameSize())
	label = ansi.Truncate(label, contentWidth, "…")
	fmt.Fprint(w, style.Render(label)) //nolint:errcheck
}

func (d sidebarListDelegate) ShortHelp() []key.Binding {
	return nil
}

func (d sidebarListDelegate) FullHelp() [][]key.Binding {
	return nil
}

func newSidebarListModel() list.Model {
	model := list.New([]list.Item{}, newSidebarListDelegate(false), 0, 0)
	model.SetShowTitle(false)
	model.SetShowFilter(false)
	model.SetFilteringEnabled(false)
	model.SetShowStatusBar(false)
	model.SetShowPagination(false)
	model.SetShowHelp(false)
	model.DisableQuitKeybindings()
	model.Styles.NoItems = styles.MutedStyle.Copy().PaddingLeft(1)
	return model
}
