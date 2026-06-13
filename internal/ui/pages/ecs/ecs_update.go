package ecs

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (p *ECSPage) OnStateChanged(state State) tea.Cmd {
	key := sessionKey(state)
	if key != p.sessionKey {
		p.sessionKey = key
		p.resetForSession()
	}
	if state.ActiveSession == nil || p.loadingClusters || p.loadedFor == key {
		return nil
	}
	p.loadingClusters = true
	p.clustersErr = ""
	return tea.Batch(p.spinner.Tick, p.loadClustersCmd(state.ActiveSession.Profile, activeRegion(state), key))
}
func (p *ECSPage) SetFocused(focused bool) tea.Cmd {
	if !focused {
		p.searchInput.Blur()
	}
	return nil
}
func (p *ECSPage) Update(msg tea.Msg, state State) tea.Cmd {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if !p.loadingClusters && !p.servicesLoading && !p.tasksLoading {
			return nil
		}
		var cmd tea.Cmd
		p.spinner, cmd = p.spinner.Update(msg)
		return cmd
	case clustersLoadedMsg:
		if msg.sessionKey != p.sessionKey {
			return nil
		}
		p.loadingClusters = false
		p.loadedFor = msg.sessionKey
		p.clustersCancel = nil
		if errors.Is(msg.err, context.Canceled) {
			return nil
		}
		if msg.err != nil {
			p.clustersErr = fmt.Sprintf("Unable to load ECS clusters: %v", msg.err)
			return nil
		}
		p.clusters = msg.clusters
		p.clustersErr = ""
		p.clusterIndex = 0
		p.syncClusterTable()
		return nil
	case servicesLoadedMsg:
		if msg.clusterARN != p.selectedCluster.ARN {
			return nil
		}
		p.servicesLoading = false
		p.servicesCancel = nil
		if errors.Is(msg.err, context.Canceled) {
			return nil
		}
		if msg.err != nil {
			p.servicesErr = fmt.Sprintf("Unable to load ECS services: %v", msg.err)
			return nil
		}
		p.services = msg.services
		p.servicesErr = ""
		p.serviceIndex = 0
		p.syncServiceTable()
		return nil
	case tasksLoadedMsg:
		if msg.clusterARN != p.selectedCluster.ARN {
			return nil
		}
		p.tasksLoading = false
		p.tasksCancel = nil
		if errors.Is(msg.err, context.Canceled) {
			return nil
		}
		if msg.err != nil {
			p.tasksErr = fmt.Sprintf("Unable to load ECS tasks: %v", msg.err)
			return nil
		}
		p.tasks = msg.tasks
		p.tasksErr = ""
		p.taskIndex = 0
		p.syncTaskTable()
		return nil
	}
	k, ok := msg.(tea.KeyMsg)
	if !ok || !state.PageFocused {
		return p.updateInput(msg)
	}
	if p.searchInput.Focused() {
		if k.Type == tea.KeyEsc {
			p.searchInput.Blur()
			return nil
		}
		if textInputKey(k) {
			cmd := p.updateInput(msg)
			p.resetFilteredCursor()
			return cmd
		}
	}
	switch p.stage {
	case ecsStageClusters:
		return p.updateClusters(k, state)
	case ecsStageResources:
		return p.updateResources(k, state)
	case ecsStageServiceDetail, ecsStageTaskDetail:
		if key.Matches(k, ecsBackKey) {
			p.stage = ecsStageResources
			return nil
		}
	}
	return nil
}
func (p *ECSPage) updateInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	p.searchInput, cmd = p.searchInput.Update(msg)
	return cmd
}
func (p *ECSPage) resetFilteredCursor() {
	switch p.stage {
	case ecsStageClusters:
		p.clusterIndex = 0
		p.clusterPaginator.Page = 0
		p.syncClusterTable()
	case ecsStageResources:
		if p.tab == ecsTabServices {
			p.serviceIndex = 0
			p.servicePaginator.Page = 0
			p.syncServiceTable()
		} else {
			p.taskIndex = 0
			p.taskPaginator.Page = 0
			p.syncTaskTable()
		}
	}
}
func (p *ECSPage) updateClusters(k tea.KeyMsg, state State) tea.Cmd {
	if key.Matches(k, ecsSearchKey) {
		return p.searchInput.Focus()
	}
	if key.Matches(k, ecsRefreshKey) && state.ActiveSession != nil {
		p.loadingClusters = true
		p.clustersErr = ""
		return tea.Batch(p.spinner.Tick, p.loadClustersCmd(state.ActiveSession.Profile, activeRegion(state), p.sessionKey))
	}
	items := p.filteredClusters()
	if p.clusterIndex >= len(items) {
		p.clusterIndex = max(0, len(items)-1)
	}
	switch {
	case key.Matches(k, ecsUpKey):
		if p.clusterIndex > 0 {
			p.clusterIndex--
		}
		p.syncClusterTable()
		return nil
	case key.Matches(k, ecsDownKey):
		if p.clusterIndex < len(items)-1 {
			p.clusterIndex++
		}
		p.syncClusterTable()
		return nil
	case key.Matches(k, ecsEnterKey):
		if len(items) > 0 && state.ActiveSession != nil {
			p.selectedCluster = items[p.clusterIndex]
			p.stage = ecsStageResources
			p.tab = ecsTabServices
			p.searchInput.SetValue("")
			p.searchInput.Blur()
			p.servicesLoading = true
			p.tasksLoading = true
			p.servicesErr = ""
			p.tasksErr = ""
			return tea.Batch(p.spinner.Tick, p.loadServicesCmd(state.ActiveSession.Profile, activeRegion(state), p.selectedCluster.ARN), p.loadTasksCmd(state.ActiveSession.Profile, activeRegion(state), p.selectedCluster.ARN))
		}
	}
	return p.updatePaged(k, true)
}
func (p *ECSPage) updateResources(k tea.KeyMsg, state State) tea.Cmd {
	if key.Matches(k, ecsBackKey) {
		p.stage = ecsStageClusters
		p.searchInput.SetValue("")
		p.searchInput.Blur()
		p.syncClusterTable()
		return nil
	}
	if key.Matches(k, ecsSearchKey) {
		return p.searchInput.Focus()
	}
	if key.Matches(k, ecsPrevTabKey) || key.Matches(k, ecsNextTabKey) {
		if p.tab == ecsTabServices {
			p.tab = ecsTabTasks
		} else {
			p.tab = ecsTabServices
		}
		p.searchInput.SetValue("")
		p.resetFilteredCursor()
		return nil
	}
	if key.Matches(k, ecsRefreshKey) && state.ActiveSession != nil {
		p.servicesLoading = true
		p.tasksLoading = true
		p.servicesErr = ""
		p.tasksErr = ""
		return tea.Batch(p.spinner.Tick, p.loadServicesCmd(state.ActiveSession.Profile, activeRegion(state), p.selectedCluster.ARN), p.loadTasksCmd(state.ActiveSession.Profile, activeRegion(state), p.selectedCluster.ARN))
	}
	if p.tab == ecsTabServices {
		return p.updateServicesTable(k)
	}
	return p.updateTasksTable(k)
}
func (p *ECSPage) updateServicesTable(k tea.KeyMsg) tea.Cmd {
	items := p.filteredServices()
	switch {
	case key.Matches(k, ecsUpKey):
		if p.serviceIndex > 0 {
			p.serviceIndex--
		}
		p.syncServiceTable()
		return nil
	case key.Matches(k, ecsDownKey):
		if p.serviceIndex < len(items)-1 {
			p.serviceIndex++
		}
		p.syncServiceTable()
		return nil
	case key.Matches(k, ecsEnterKey):
		if len(items) > 0 {
			p.selectedService = items[p.serviceIndex]
			p.stage = ecsStageServiceDetail
		}
		return nil
	}
	return p.updatePaged(k, false)
}
func (p *ECSPage) updateTasksTable(k tea.KeyMsg) tea.Cmd {
	items := p.filteredTasks()
	switch {
	case key.Matches(k, ecsUpKey):
		if p.taskIndex > 0 {
			p.taskIndex--
		}
		p.syncTaskTable()
		return nil
	case key.Matches(k, ecsDownKey):
		if p.taskIndex < len(items)-1 {
			p.taskIndex++
		}
		p.syncTaskTable()
		return nil
	case key.Matches(k, ecsEnterKey):
		if len(items) > 0 {
			p.selectedTask = items[p.taskIndex]
			p.stage = ecsStageTaskDetail
		}
		return nil
	}
	return p.updatePaged(k, false)
}
func (p *ECSPage) updatePaged(k tea.KeyMsg, clusters bool) tea.Cmd {
	if clusters {
		prev := p.clusterPaginator.Page
		var cmd tea.Cmd
		p.clusterPaginator, cmd = p.clusterPaginator.Update(k)
		if p.clusterPaginator.Page != prev {
			start, _ := p.clusterPaginator.GetSliceBounds(len(p.filteredClusters()))
			p.clusterIndex = start
			p.syncClusterTable()
		}
		return cmd
	}
	if p.tab == ecsTabServices {
		prev := p.servicePaginator.Page
		var cmd tea.Cmd
		p.servicePaginator, cmd = p.servicePaginator.Update(k)
		if p.servicePaginator.Page != prev {
			start, _ := p.servicePaginator.GetSliceBounds(len(p.filteredServices()))
			p.serviceIndex = start
			p.syncServiceTable()
		}
		return cmd
	}
	prev := p.taskPaginator.Page
	var cmd tea.Cmd
	p.taskPaginator, cmd = p.taskPaginator.Update(k)
	if p.taskPaginator.Page != prev {
		start, _ := p.taskPaginator.GetSliceBounds(len(p.filteredTasks()))
		p.taskIndex = start
		p.syncTaskTable()
	}
	return cmd
}
