package ecs

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	domainecs "aws-terminal/internal/domain/ecs"
	domainsession "aws-terminal/internal/domain/session"
)

type fakeECSService struct{}

func (fakeECSService) ListClusters(context.Context, string, string) ([]domainecs.Cluster, error) {
	return nil, nil
}
func (fakeECSService) ListServices(context.Context, string, string, string) ([]domainecs.Service, error) {
	return nil, nil
}
func (fakeECSService) ListTasks(context.Context, string, string, string) ([]domainecs.Task, error) {
	return nil, nil
}

func testState() State {
	return State{ActiveSession: &domainsession.Session{Profile: "dev", Region: "eu-west-1"}, SelectedRegion: "eu-west-1", PageFocused: true}
}

func TestSearchInputAcceptsKeybindLetters(t *testing.T) {
	p := NewECSPage(fakeECSService{})
	p.stage = ecsStageClusters
	if cmd := p.Update(tea.KeyMsg{Type: tea.KeyCtrlF}, testState()); cmd == nil {
		t.Fatal("expected focus command")
	}
	p.searchInput.Focus()
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}, testState())
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}, testState())
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")}, testState())
	if got := p.searchInput.Value(); got != "rqb" {
		t.Fatalf("search value = %q", got)
	}
}

func TestSelectingClusterStartsServicesAndTasksLoads(t *testing.T) {
	p := NewECSPage(fakeECSService{})
	p.clusters = []domainecs.Cluster{{Name: "prod", ARN: "arn:cluster"}}
	p.syncClusterTable()
	cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter}, testState())
	if p.stage != ecsStageResources {
		t.Fatalf("stage = %v", p.stage)
	}
	if !p.servicesLoading || !p.tasksLoading {
		t.Fatalf("services/tasks should be loading")
	}
	if cmd == nil {
		t.Fatal("expected load command")
	}
}

func TestSwitchTabsAndOpenDetails(t *testing.T) {
	p := NewECSPage(fakeECSService{})
	p.stage = ecsStageResources
	p.tab = ecsTabServices
	p.services = []domainecs.Service{{Name: "api", ARN: "svc"}}
	p.tasks = []domainecs.Task{{ID: "task", ARN: "task"}}
	p.syncServiceTable()
	p.syncTaskTable()
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")}, testState())
	if p.tab != ecsTabTasks {
		t.Fatalf("tab = %v", p.tab)
	}
	p.Update(tea.KeyMsg{Type: tea.KeyEnter}, testState())
	if p.stage != ecsStageTaskDetail {
		t.Fatalf("stage = %v", p.stage)
	}
	p.Update(tea.KeyMsg{Type: tea.KeyEsc}, testState())
	if p.stage != ecsStageResources {
		t.Fatalf("stage after back = %v", p.stage)
	}
}
