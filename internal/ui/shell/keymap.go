package shell

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Focus     key.Binding
	BackFocus key.Binding
	Select    key.Binding
	Refresh   key.Binding
	Palette   key.Binding
	Quit      key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Focus, k.BackFocus, k.Select, k.Palette, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Up, k.Down}, {k.Focus, k.BackFocus, k.Select, k.Refresh, k.Palette}, {k.Quit}}
}

var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Focus: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next focus"),
	),
	BackFocus: key.NewBinding(
		key.WithKeys("shift+tab", "backtab"),
		key.WithHelp("shift+tab", "prev focus"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "apply / open"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh profiles"),
	),
	Palette: key.NewBinding(
		key.WithKeys(":", "ctrl+p"),
		key.WithHelp(":", "commands"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
