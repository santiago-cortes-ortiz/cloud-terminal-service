package ecr

import "github.com/charmbracelet/bubbles/key"

var (
	ecrUpKey       = key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up"))
	ecrDownKey     = key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down"))
	ecrEnterKey    = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select/continue"))
	ecrBackKey     = key.NewBinding(key.WithKeys("b", "esc"), key.WithHelp("b/esc", "back"))
	ecrCreateKey   = key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new repo"))
	ecrRefreshKey  = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh"))
	ecrSearchKey   = key.NewBinding(key.WithKeys("ctrl+f"), key.WithHelp("ctrl+f", "search"))
	ecrPagePrevKey = key.NewBinding(key.WithKeys("left", "h", "pgup"), key.WithHelp("←/h", "prev page"))
	ecrPageNextKey = key.NewBinding(key.WithKeys("right", "l", "pgdown"), key.WithHelp("→/l", "next page"))
	ecrManualKey   = key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "manual image"))
	ecrTabHelpKey  = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "sidebar"))
)
