package ecr

import (
	"context"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"

	appsecr "aws-terminal/internal/application/ecr"
	domainecr "aws-terminal/internal/domain/ecr"
	"aws-terminal/internal/ui/styles"
)

type ECRService interface {
	ListRepositories(ctx context.Context, profileName, region string) ([]domainecr.Repository, error)
	CreateRepository(ctx context.Context, input domainecr.CreateRepositoryInput) (domainecr.Repository, error)
	ListImages(ctx context.Context, profileName, region, repositoryName string) ([]domainecr.RepositoryImage, error)
	ListLocalImages(ctx context.Context) ([]domainecr.LocalImage, error)
	BuildPushPlan(input appsecr.BuildPushPlanInput) (domainecr.PushPlan, error)
	ExecutePush(ctx context.Context, plan domainecr.PushPlan, progress chan<- domainecr.PushProgress) (domainecr.PushResult, error)
}

type ecrStage int

const (
	ecrStageRepository ecrStage = iota
	ecrStageCreateRepository
	ecrStageRepositoryImages
	ecrStageLocalImage
	ecrStageTag
	ecrStageReview
	ecrStagePush
)

type repositoriesLoadedMsg struct {
	sessionKey   string
	repositories []domainecr.Repository
	err          error
}
type repositoryCreatedMsg struct {
	repository domainecr.Repository
	err        error
}
type repositoryImagesLoadedMsg struct {
	repositoryName string
	images         []domainecr.RepositoryImage
	err            error
}
type localImagesLoadedMsg struct {
	images []domainecr.LocalImage
	err    error
}
type pushPlanBuiltMsg struct {
	plan domainecr.PushPlan
	err  error
}
type pushStartedMsg struct{ events <-chan pushEvent }
type pushEventMsg struct{ event pushEvent }

type pushEvent struct {
	progress *domainecr.PushProgress
	result   *domainecr.PushResult
	err      error
	done     bool
}

func (repositoriesLoadedMsg) OwnerPageID() string     { return "ecr" }
func (repositoryCreatedMsg) OwnerPageID() string      { return "ecr" }
func (repositoryImagesLoadedMsg) OwnerPageID() string { return "ecr" }
func (localImagesLoadedMsg) OwnerPageID() string      { return "ecr" }
func (pushPlanBuiltMsg) OwnerPageID() string          { return "ecr" }
func (pushStartedMsg) OwnerPageID() string            { return "ecr" }
func (pushEventMsg) OwnerPageID() string              { return "ecr" }

type ECRPage struct {
	service             ECRService
	stage               ecrStage
	sessionKey          string
	loadedFor           string
	loadingRepositories bool
	repositoryErr       string
	repositories        []domainecr.Repository
	repositoryIndex     int
	selectedRepository  domainecr.Repository
	searchInput         textinput.Model
	createInput         textinput.Model
	imagesLoading       bool
	imagesErr           string
	repositoryImages    []domainecr.RepositoryImage
	imageTable          table.Model
	imagePaginator      paginator.Model
	localLoading        bool
	localErr            string
	localImages         []domainecr.LocalImage
	localIndex          int
	localTable          table.Model
	localPaginator      paginator.Model
	manualInput         textinput.Model
	tagInput            textinput.Model
	planning            bool
	planErr             string
	plan                *domainecr.PushPlan
	pushing             bool
	pushErr             string
	pushMessage         string
	pushProgress        domainecr.PushProgress
	pushResult          *domainecr.PushResult
	pushEvents          <-chan pushEvent
	spinner             spinner.Model
	pushBar             progress.Model
	loadCancel          context.CancelFunc
	createCancel        context.CancelFunc
	imagesCancel        context.CancelFunc
	localCancel         context.CancelFunc
	pushCancel          context.CancelFunc
}

func NewECRPage(service ECRService) *ECRPage {
	search := textinput.New()
	search.Prompt = "Search: "
	search.Placeholder = "repository name"
	search.CharLimit = 256
	create := textinput.New()
	create.Prompt = "Repository name: "
	create.Placeholder = "my-service"
	create.CharLimit = 256
	manual := textinput.New()
	manual.Prompt = "Image: "
	manual.Placeholder = "local-image:tag"
	manual.CharLimit = 512
	tag := textinput.New()
	tag.Prompt = "Destination tag: "
	tag.Placeholder = "latest"
	tag.CharLimit = 128
	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = styles.StatusStyle

	imageTable := table.New(
		table.WithColumns(ecrImageTableColumns()),
		table.WithHeight(9),
	)
	imageTable.SetStyles(ecrTableStyles())
	imagePaginator := paginator.New(paginator.WithPerPage(8))
	imagePaginator.Type = paginator.Arabic

	localTable := table.New(
		table.WithColumns(ecrLocalImageTableColumns()),
		table.WithHeight(9),
	)
	localTable.SetStyles(ecrTableStyles())
	localPaginator := paginator.New(paginator.WithPerPage(8))
	localPaginator.Type = paginator.Arabic

	return &ECRPage{service: service, stage: ecrStageRepository, searchInput: search, createInput: create, manualInput: manual, tagInput: tag, imageTable: imageTable, imagePaginator: imagePaginator, localTable: localTable, localPaginator: localPaginator, spinner: spin, pushBar: progress.New(progress.WithScaledGradient("#5A56E0", "#EE6FF8"))}
}

func (*ECRPage) ID() string    { return "ecr" }
func (*ECRPage) Title() string { return "ECR" }
func (*ECRPage) Description() string {
	return "Search private ECR repositories, view images, and push local Docker images."
}
