package cloudfront

import "github.com/charmbracelet/bubbles/key"

var (
	cloudFrontUpKey     = key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up"))
	cloudFrontDownKey   = key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down"))
	cloudFrontEnterKey  = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "continue/create"))
	cloudFrontBackKey   = key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "back"))
	cloudFrontCancelKey = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel/back"))
	cloudFrontCopyKey   = key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy cli cmd"))
	cloudFrontTabKey    = key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("tab/shift+tab", "switch focus"))
)
