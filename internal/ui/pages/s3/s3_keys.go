package s3

import "github.com/charmbracelet/bubbles/key"

var (
	s3BucketUpKey     = key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up"))
	s3BucketDownKey   = key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down"))
	s3EnterKey        = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "continue"))
	s3ToggleKey       = key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle delete"))
	s3OptimizeKey     = key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "toggle planning"))
	s3MetadataKey     = key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "metadata preset"))
	s3BackKey         = key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "back"))
	s3CancelKey       = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel/back"))
	s3RefreshKey      = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh"))
	s3ContinueKey     = key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "continue"))
	s3ViewportUpKey   = key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "scroll up"))
	s3ViewportDownKey = key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "scroll down"))
	s3PageUpKey       = key.NewBinding(key.WithKeys("pgup", "ctrl+u"), key.WithHelp("pgup", "page up"))
	s3PageDownKey     = key.NewBinding(key.WithKeys("pgdown", "ctrl+d"), key.WithHelp("pgdn", "page down"))
	s3PickerOpenKey   = key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "open"))
	s3PickerSelectKey = key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("enter", "select"))
	s3PickerBackKey   = key.NewBinding(key.WithKeys("backspace"), key.WithHelp("⌫/h", "back"))
	s3InvalidateKey   = key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "cloudfront"))
	s3TabHelpKey      = key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("tab/shift+tab", "switch focus"))
)
