package ecs

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	domainecs "aws-terminal/internal/domain/ecs"
	"aws-terminal/internal/ui/styles"
	"aws-terminal/internal/ui/workflow"
)

func sessionKey(state State) string   { return workflow.SessionKey(state) }
func activeRegion(state State) string { return workflow.ActiveRegion(state) }

func tableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.Bold(true).Foreground(lipgloss.Color("39"))
	s.Selected = styles.FocusedSelectedSidebarItemStyle
	return s
}
func clusterColumns() []table.Column {
	return []table.Column{{Title: "Cluster", Width: 28}, {Title: "Status", Width: 10}, {Title: "Services", Width: 8}, {Title: "Running", Width: 8}, {Title: "Pending", Width: 8}, {Title: "Instances", Width: 9}}
}
func serviceColumns() []table.Column {
	return []table.Column{{Title: "Service", Width: 28}, {Title: "Status", Width: 10}, {Title: "Task definition", Width: 22}, {Title: "Tasks", Width: 16}, {Title: "Created", Width: 16}}
}
func taskColumns() []table.Column {
	return []table.Column{{Title: "Task", Width: 13}, {Title: "Last", Width: 9}, {Title: "Desired", Width: 8}, {Title: "Task definition", Width: 18}, {Title: "IP", Width: 15}, {Title: "Created", Width: 12}, {Title: "Started", Width: 12}}
}

func textInputKey(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyRunes, tea.KeySpace, tea.KeyBackspace, tea.KeyDelete:
		return true
	default:
		return false
	}
}
func lowerContains(v, q string) bool { return strings.Contains(strings.ToLower(v), strings.ToLower(q)) }
func timeText(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Local().Format("2006-01-02 15:04")
}

func tableTimeText(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Local().Format("01-02 15:04")
}

func value(v string) string {
	if strings.TrimSpace(v) == "" {
		return "—"
	}
	return v
}
func taskCount(s domainecs.Service) string {
	base := fmt.Sprintf("%d/%d", s.RunningCount, s.DesiredCount)
	if s.PendingCount > 0 {
		base += fmt.Sprintf(" (+%d pending)", s.PendingCount)
	}
	return base
}

func (p *ECSPage) filteredClusters() []domainecs.Cluster {
	q := strings.TrimSpace(p.searchInput.Value())
	if q == "" {
		return p.clusters
	}
	out := []domainecs.Cluster{}
	for _, c := range p.clusters {
		if lowerContains(c.Name, q) || lowerContains(c.Status, q) {
			out = append(out, c)
		}
	}
	return out
}
func (p *ECSPage) filteredServices() []domainecs.Service {
	q := strings.TrimSpace(p.searchInput.Value())
	if q == "" {
		return p.services
	}
	out := []domainecs.Service{}
	for _, s := range p.services {
		if lowerContains(s.Name, q) || lowerContains(s.Status, q) || lowerContains(s.TaskDefinition, q) {
			out = append(out, s)
		}
	}
	return out
}
func (p *ECSPage) filteredTasks() []domainecs.Task {
	q := strings.TrimSpace(p.searchInput.Value())
	if q == "" {
		return p.tasks
	}
	out := []domainecs.Task{}
	for _, t := range p.tasks {
		if lowerContains(t.ID, q) || lowerContains(t.LastStatus, q) || lowerContains(t.DesiredStatus, q) || lowerContains(t.TaskDefinition, q) || lowerContains(t.PrivateIP, q) {
			out = append(out, t)
		}
	}
	return out
}

func (p *ECSPage) syncClusterTable() {
	items := p.filteredClusters()
	p.clusterPaginator.SetTotalPages(len(items))
	if p.clusterPaginator.Page >= p.clusterPaginator.TotalPages {
		p.clusterPaginator.Page = max(0, p.clusterPaginator.TotalPages-1)
	}
	if p.clusterIndex >= len(items) {
		p.clusterIndex = max(0, len(items)-1)
	}
	if len(items) > 0 {
		p.clusterPaginator.Page = p.clusterIndex / max(1, p.clusterPaginator.PerPage)
	}
	start, end := p.clusterPaginator.GetSliceBounds(len(items))
	if p.clusterIndex < start || p.clusterIndex >= end {
		p.clusterIndex = start
	}
	rows := []table.Row{}
	for _, c := range items[start:end] {
		rows = append(rows, table.Row{c.Name, c.Status, fmt.Sprint(c.ActiveServicesCount), fmt.Sprint(c.RunningTasksCount), fmt.Sprint(c.PendingTasksCount), fmt.Sprint(c.RegisteredInstanceCount)})
	}
	p.clusterTable.SetRows(rows)
	p.clusterTable.SetCursor(max(0, p.clusterIndex-start))
}
func (p *ECSPage) syncServiceTable() {
	items := p.filteredServices()
	p.servicePaginator.SetTotalPages(len(items))
	if p.servicePaginator.Page >= p.servicePaginator.TotalPages {
		p.servicePaginator.Page = max(0, p.servicePaginator.TotalPages-1)
	}
	if p.serviceIndex >= len(items) {
		p.serviceIndex = max(0, len(items)-1)
	}
	if len(items) > 0 {
		p.servicePaginator.Page = p.serviceIndex / max(1, p.servicePaginator.PerPage)
	}
	start, end := p.servicePaginator.GetSliceBounds(len(items))
	if p.serviceIndex < start || p.serviceIndex >= end {
		p.serviceIndex = start
	}
	rows := []table.Row{}
	for _, s := range items[start:end] {
		rows = append(rows, table.Row{s.Name, s.Status, value(s.TaskDefinition), taskCount(s), timeText(s.CreatedAt)})
	}
	p.serviceTable.SetRows(rows)
	p.serviceTable.SetCursor(max(0, p.serviceIndex-start))
}
func (p *ECSPage) syncTaskTable() {
	items := p.filteredTasks()
	p.taskPaginator.SetTotalPages(len(items))
	if p.taskPaginator.Page >= p.taskPaginator.TotalPages {
		p.taskPaginator.Page = max(0, p.taskPaginator.TotalPages-1)
	}
	if p.taskIndex >= len(items) {
		p.taskIndex = max(0, len(items)-1)
	}
	if len(items) > 0 {
		p.taskPaginator.Page = p.taskIndex / max(1, p.taskPaginator.PerPage)
	}
	start, end := p.taskPaginator.GetSliceBounds(len(items))
	if p.taskIndex < start || p.taskIndex >= end {
		p.taskIndex = start
	}
	rows := []table.Row{}
	for _, t := range items[start:end] {
		rows = append(rows, table.Row{shortText(t.ID, 13), t.LastStatus, t.DesiredStatus, value(t.TaskDefinition), value(t.PrivateIP), tableTimeText(t.CreatedAt), tableTimeText(t.StartedAt)})
	}
	p.taskTable.SetRows(rows)
	p.taskTable.SetCursor(max(0, p.taskIndex-start))
}

func shortText(v string, width int) string {
	if width <= 1 || len([]rune(v)) <= width {
		return v
	}
	runes := []rune(v)
	return string(runes[:width-1]) + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
