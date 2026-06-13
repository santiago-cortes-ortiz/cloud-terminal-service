package ecs

import (
	"context"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"

	domainecs "aws-terminal/internal/domain/ecs"
	"aws-terminal/internal/ui/styles"
)

type ECSService interface {
	ListClusters(ctx context.Context, profileName, region string) ([]domainecs.Cluster, error)
	ListServices(ctx context.Context, profileName, region, clusterARN string) ([]domainecs.Service, error)
	ListTasks(ctx context.Context, profileName, region, clusterARN string) ([]domainecs.Task, error)
}

type ecsStage int

const (
	ecsStageClusters ecsStage = iota
	ecsStageResources
	ecsStageServiceDetail
	ecsStageTaskDetail
)

type ecsTab int

const (
	ecsTabServices ecsTab = iota
	ecsTabTasks
)

type clustersLoadedMsg struct {
	sessionKey string
	clusters   []domainecs.Cluster
	err        error
}
type servicesLoadedMsg struct {
	clusterARN string
	services   []domainecs.Service
	err        error
}
type tasksLoadedMsg struct {
	clusterARN string
	tasks      []domainecs.Task
	err        error
}

func (clustersLoadedMsg) OwnerPageID() string { return "ecs" }
func (servicesLoadedMsg) OwnerPageID() string { return "ecs" }
func (tasksLoadedMsg) OwnerPageID() string    { return "ecs" }

type ECSPage struct {
	service          ECSService
	stage            ecsStage
	tab              ecsTab
	sessionKey       string
	loadedFor        string
	loadingClusters  bool
	clustersErr      string
	clusters         []domainecs.Cluster
	clusterIndex     int
	selectedCluster  domainecs.Cluster
	searchInput      textinput.Model
	clusterTable     table.Model
	clusterPaginator paginator.Model
	servicesLoading  bool
	servicesErr      string
	services         []domainecs.Service
	serviceIndex     int
	serviceTable     table.Model
	servicePaginator paginator.Model
	selectedService  domainecs.Service
	tasksLoading     bool
	tasksErr         string
	tasks            []domainecs.Task
	taskIndex        int
	taskTable        table.Model
	taskPaginator    paginator.Model
	selectedTask     domainecs.Task
	spinner          spinner.Model
	clustersCancel   context.CancelFunc
	servicesCancel   context.CancelFunc
	tasksCancel      context.CancelFunc
}

func NewECSPage(service ECSService) *ECSPage {
	search := textinput.New()
	search.Prompt = "Search: "
	search.Placeholder = "filter"
	search.CharLimit = 256
	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = styles.StatusStyle
	ct := table.New(table.WithColumns(clusterColumns()), table.WithHeight(9))
	ct.SetStyles(tableStyles())
	st := table.New(table.WithColumns(serviceColumns()), table.WithHeight(9))
	st.SetStyles(tableStyles())
	tt := table.New(table.WithColumns(taskColumns()), table.WithHeight(9))
	tt.SetStyles(tableStyles())
	cp := paginator.New(paginator.WithPerPage(8))
	cp.Type = paginator.Arabic
	sp := paginator.New(paginator.WithPerPage(8))
	sp.Type = paginator.Arabic
	tp := paginator.New(paginator.WithPerPage(8))
	tp.Type = paginator.Arabic
	return &ECSPage{service: service, stage: ecsStageClusters, searchInput: search, spinner: spin, clusterTable: ct, clusterPaginator: cp, serviceTable: st, servicePaginator: sp, taskTable: tt, taskPaginator: tp}
}

func (*ECSPage) ID() string              { return "ecs" }
func (*ECSPage) Title() string           { return "ECS" }
func (*ECSPage) Description() string     { return "Browse ECS clusters, services, and tasks." }
func (p *ECSPage) HasFocusedInput() bool { return p.searchInput.Focused() }
