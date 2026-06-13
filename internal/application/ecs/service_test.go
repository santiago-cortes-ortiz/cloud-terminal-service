package ecs

import (
	"context"
	"reflect"
	"testing"
	"time"

	domainecs "aws-terminal/internal/domain/ecs"
)

type fakeAPI struct {
	clusters []domainecs.Cluster
	services []domainecs.Service
	tasks    []domainecs.Task
}

func (f fakeAPI) ListClusters(context.Context, string, string) ([]domainecs.Cluster, error) {
	return append([]domainecs.Cluster(nil), f.clusters...), nil
}
func (f fakeAPI) ListServices(context.Context, string, string, string) ([]domainecs.Service, error) {
	return append([]domainecs.Service(nil), f.services...), nil
}
func (f fakeAPI) ListTasks(context.Context, string, string, string) ([]domainecs.Task, error) {
	return append([]domainecs.Task(nil), f.tasks...), nil
}

func TestListClustersValidatesProfileAndSortsActiveFirst(t *testing.T) {
	svc := NewService(fakeAPI{clusters: []domainecs.Cluster{{Name: "z", Status: "INACTIVE"}, {Name: "b", Status: "ACTIVE"}, {Name: "a", Status: "ACTIVE"}}})
	if _, err := svc.ListClusters(context.Background(), " ", "eu-west-1"); err == nil {
		t.Fatal("expected profile validation error")
	}
	got, err := svc.ListClusters(context.Background(), "dev", "eu-west-1")
	if err != nil {
		t.Fatal(err)
	}
	names := []string{got[0].Name, got[1].Name, got[2].Name}
	if !reflect.DeepEqual(names, []string{"a", "b", "z"}) {
		t.Fatalf("unexpected order: %v", names)
	}
}

func TestListServicesValidatesInputsAndSortsActiveFirst(t *testing.T) {
	svc := NewService(fakeAPI{services: []domainecs.Service{{Name: "z", Status: "DRAINING"}, {Name: "b", Status: "ACTIVE"}, {Name: "a", Status: "ACTIVE"}}})
	if _, err := svc.ListServices(context.Background(), "dev", "eu-west-1", " "); err == nil {
		t.Fatal("expected cluster ARN validation error")
	}
	got, err := svc.ListServices(context.Background(), "dev", "eu-west-1", "cluster")
	if err != nil {
		t.Fatal(err)
	}
	names := []string{got[0].Name, got[1].Name, got[2].Name}
	if !reflect.DeepEqual(names, []string{"a", "b", "z"}) {
		t.Fatalf("unexpected order: %v", names)
	}
}

func TestListTasksFiltersStoppedAndSortsNonRunningNewestFirst(t *testing.T) {
	older := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	svc := NewService(fakeAPI{tasks: []domainecs.Task{{ID: "run", LastStatus: "RUNNING", CreatedAt: newer}, {ID: "stop", LastStatus: "STOPPED", CreatedAt: newer}, {ID: "pend-old", LastStatus: "PENDING", CreatedAt: older}, {ID: "pend-new", LastStatus: "PENDING", CreatedAt: newer}}})
	got, err := svc.ListTasks(context.Background(), "dev", "eu-west-1", "cluster")
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(got))
	for i := range got {
		ids[i] = got[i].ID
	}
	want := []string{"pend-new", "pend-old", "run"}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("got %v want %v", ids, want)
	}
}
