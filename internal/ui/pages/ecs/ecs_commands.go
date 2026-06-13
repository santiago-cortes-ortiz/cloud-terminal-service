package ecs

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	domainecs "aws-terminal/internal/domain/ecs"
)

func (p *ECSPage) loadClustersCmd(profile, region, key string) tea.Cmd {
	if p.clustersCancel != nil {
		p.clustersCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	p.clustersCancel = cancel
	return func() tea.Msg {
		clusters, err := p.service.ListClusters(ctx, profile, region)
		return clustersLoadedMsg{sessionKey: key, clusters: clusters, err: err}
	}
}
func (p *ECSPage) loadServicesCmd(profile, region, clusterARN string) tea.Cmd {
	if p.servicesCancel != nil {
		p.servicesCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	p.servicesCancel = cancel
	return func() tea.Msg {
		services, err := p.service.ListServices(ctx, profile, region, clusterARN)
		return servicesLoadedMsg{clusterARN: clusterARN, services: services, err: err}
	}
}
func (p *ECSPage) loadTasksCmd(profile, region, clusterARN string) tea.Cmd {
	if p.tasksCancel != nil {
		p.tasksCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	p.tasksCancel = cancel
	return func() tea.Msg {
		tasks, err := p.service.ListTasks(ctx, profile, region, clusterARN)
		return tasksLoadedMsg{clusterARN: clusterARN, tasks: tasks, err: err}
	}
}
func (p *ECSPage) cancelResourceLoads() {
	if p.servicesCancel != nil {
		p.servicesCancel()
		p.servicesCancel = nil
	}
	if p.tasksCancel != nil {
		p.tasksCancel()
		p.tasksCancel = nil
	}
}
func (p *ECSPage) resetForSession() {
	if p.clustersCancel != nil {
		p.clustersCancel()
		p.clustersCancel = nil
	}
	p.cancelResourceLoads()
	p.loadedFor = ""
	p.loadingClusters = false
	p.clustersErr = ""
	p.clusters = nil
	p.clusterIndex = 0
	p.stage = ecsStageClusters
	p.selectedCluster = domainecs.Cluster{}
	p.services = nil
	p.tasks = nil
	p.searchInput.SetValue("")
}
