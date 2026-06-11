package pageapi

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	domainauth "aws-terminal/internal/domain/auth"
	domainprofile "aws-terminal/internal/domain/profile"
	domainsession "aws-terminal/internal/domain/session"
)

type State struct {
	HighlightedProfile *domainprofile.Profile
	ActiveSession      *domainsession.Session
	SelectedRegion     string
	StatusMessage      string
	ErrorMessage       string
	ProfileBusy        bool
	AuthPrompt         *domainauth.Prompt
	PageFocused        bool
}

type OpenPageMsg struct {
	PageID string
	Focus  bool
}

// OwnedMsg identifies an asynchronous message that belongs to a specific page.
// The shell uses this to route in-flight command results to the page that
// started them, even if the user has navigated to another page meanwhile.
type OwnedMsg interface {
	OwnerPageID() string
}

// Status captures page-local workflow status separately from global shell
// status such as profile loading/authentication.
type Status struct {
	Message string
	Error   string
}

// StatusProvider is optional. Pages can implement it when they have local
// workflow status that should be surfaced by the shell footer.
type StatusProvider interface {
	PageStatus(state State) Status
}

type Page interface {
	ID() string
	Title() string
	Description() string
	OnStateChanged(state State) tea.Cmd
	SetFocused(focused bool) tea.Cmd
	Update(msg tea.Msg, state State) tea.Cmd
	View(state State, width, height int) string
	ShortHelp() []key.Binding
	FullHelp() [][]key.Binding
}
