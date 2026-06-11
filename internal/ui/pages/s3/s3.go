package s3

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"

	apps3 "aws-terminal/internal/application/s3"
	"aws-terminal/internal/config"
	domains3 "aws-terminal/internal/domain/s3"
)

type S3Service interface {
	ListBuckets(ctx context.Context, profileName, region string) ([]domains3.Bucket, error)
	InspectSource(sourcePath string) (domains3.SourceSelection, error)
	BuildSyncPlan(ctx context.Context, input apps3.BuildSyncPlanInput) (domains3.SyncPlan, error)
	ExecuteSync(ctx context.Context, plan domains3.SyncPlan, progress chan<- domains3.SyncProgress) (domains3.SyncResult, error)
}

type s3Stage int

const (
	s3StageBucket s3Stage = iota
	s3StageSource
	s3StagePrefix
	s3StageReview
	s3StageConfirmDelete
	s3StageSync
)

type s3BucketsLoadedMsg struct {
	sessionKey string
	buckets    []domains3.Bucket
	err        error
}

type s3SourceInspectedMsg struct {
	source domains3.SourceSelection
	err    error
}

type s3SyncPlanBuiltMsg struct {
	plan domains3.SyncPlan
	err  error
}

type s3SyncStartedMsg struct {
	events <-chan s3SyncEvent
}

type s3SyncEventMsg struct {
	event s3SyncEvent
}

type s3SyncEvent struct {
	progress *domains3.SyncProgress
	result   *domains3.SyncResult
	err      error
	done     bool
}

type s3ClearSyncMessageMsg struct {
	seq int
}

func (s3BucketsLoadedMsg) OwnerPageID() string    { return "s3-buckets" }
func (s3SourceInspectedMsg) OwnerPageID() string  { return "s3-buckets" }
func (s3SyncPlanBuiltMsg) OwnerPageID() string    { return "s3-buckets" }
func (s3SyncStartedMsg) OwnerPageID() string      { return "s3-buckets" }
func (s3SyncEventMsg) OwnerPageID() string        { return "s3-buckets" }
func (s3ClearSyncMessageMsg) OwnerPageID() string { return "s3-buckets" }

type S3Page struct {
	service             S3Service
	stage               s3Stage
	sessionKey          string
	bucketsLoadedFor    string
	loadingBuckets      bool
	bucketErr           string
	buckets             []domains3.Bucket
	bucketIndex         int
	selectedBucket      string
	picker              filepicker.Model
	syncBar             progress.Model
	reviewViewport      viewport.Model
	sourceInfo          *domains3.SourceSelection
	inspectingSource    bool
	sourceErr           string
	prefixInput         textinput.Model
	confirmInput        textinput.Model
	deleteEnabled       bool
	optimizedPlanning   bool
	staticWebsitePreset bool
	planning            bool
	plan                *domains3.SyncPlan
	planErr             string
	syncing             bool
	syncErr             string
	syncMessage         string
	offerInvalidation   bool
	syncMessageSeq      int
	syncProgress        domains3.SyncProgress
	syncResult          *domains3.SyncResult
	syncEvents          <-chan s3SyncEvent
	loadBucketsCancel   context.CancelFunc
	inspectCancel       context.CancelFunc
	planCancel          context.CancelFunc
	syncCancel          context.CancelFunc
	lastSyncStartedAt   time.Time
	lastSyncFinishedAt  time.Time
	preferenceStore     config.PreferenceStore
	preferences         config.Preferences
}

func NewS3Page(service S3Service) *S3Page {
	return NewS3PageWithPreferences(service, nil)
}

func NewS3PageWithPreferences(service S3Service, preferenceStore config.PreferenceStore) *S3Page {
	preferences := config.Preferences{}
	if preferenceStore != nil {
		if loaded, err := preferenceStore.Load(); err == nil {
			preferences = loaded
		}
	}

	picker := filepicker.New()
	picker.CurrentDirectory = preferredFilePickerDirectory(preferences.S3SourceDirectory)
	picker.ShowHidden = false
	picker.FileAllowed = true
	picker.DirAllowed = true
	picker.ShowPermissions = false
	picker.AutoHeight = false
	picker.KeyMap.Open = key.NewBinding(
		key.WithKeys("right", "l", "enter", " "),
		key.WithHelp("→/l", "open"),
	)
	picker.KeyMap.Select = key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter", "select"),
	)
	picker.KeyMap.Back = key.NewBinding(
		key.WithKeys("backspace", "left", "h"),
		key.WithHelp("backspace", "back"),
	)

	prefixInput := textinput.New()
	prefixInput.Prompt = "S3 prefix: "
	prefixInput.Placeholder = "optional/path/in/bucket"
	prefixInput.CharLimit = 1024

	confirmInput := textinput.New()
	confirmInput.Prompt = "Type DELETE: "
	confirmInput.Placeholder = "DELETE"
	confirmInput.CharLimit = len(deleteConfirmationText)

	syncBar := progress.New(progress.WithScaledGradient("#5A56E0", "#EE6FF8"))
	reviewViewport := viewport.New(0, 0)

	return &S3Page{
		service:         service,
		stage:           s3StageBucket,
		picker:          picker,
		syncBar:         syncBar,
		reviewViewport:  reviewViewport,
		prefixInput:     prefixInput,
		confirmInput:    confirmInput,
		deleteEnabled:   false,
		preferenceStore: preferenceStore,
		preferences:     preferences,
	}
}

func (*S3Page) ID() string {
	return "s3-buckets"
}

func (*S3Page) Title() string {
	return "S3 Buckets"
}

func (*S3Page) Description() string {
	return "Browse buckets and sync local files or folders with confirmation."
}
