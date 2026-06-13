package ecs

import (
	"context"
	"fmt"
	"sort"
	"strings"

	domainecs "aws-terminal/internal/domain/ecs"
)

type Service struct{ api API }

func NewService(api API) *Service { return &Service{api: api} }

func (s *Service) ListClusters(ctx context.Context, profileName, region string) ([]domainecs.Cluster, error) {
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		return nil, fmt.Errorf("profile name is required")
	}
	clusters, err := s.api.ListClusters(ctx, profileName, strings.TrimSpace(region))
	if err != nil {
		return nil, err
	}
	SortClusters(clusters)
	return clusters, nil
}

func (s *Service) ListServices(ctx context.Context, profileName, region, clusterARN string) ([]domainecs.Service, error) {
	profileName = strings.TrimSpace(profileName)
	clusterARN = strings.TrimSpace(clusterARN)
	if profileName == "" {
		return nil, fmt.Errorf("profile name is required")
	}
	if clusterARN == "" {
		return nil, fmt.Errorf("cluster ARN is required")
	}
	services, err := s.api.ListServices(ctx, profileName, strings.TrimSpace(region), clusterARN)
	if err != nil {
		return nil, err
	}
	SortServices(services)
	return services, nil
}

func (s *Service) ListTasks(ctx context.Context, profileName, region, clusterARN string) ([]domainecs.Task, error) {
	profileName = strings.TrimSpace(profileName)
	clusterARN = strings.TrimSpace(clusterARN)
	if profileName == "" {
		return nil, fmt.Errorf("profile name is required")
	}
	if clusterARN == "" {
		return nil, fmt.Errorf("cluster ARN is required")
	}
	tasks, err := s.api.ListTasks(ctx, profileName, strings.TrimSpace(region), clusterARN)
	if err != nil {
		return nil, err
	}
	filtered := tasks[:0]
	for _, task := range tasks {
		if !strings.EqualFold(strings.TrimSpace(task.LastStatus), "STOPPED") {
			filtered = append(filtered, task)
		}
	}
	SortTasks(filtered)
	return filtered, nil
}

func SortClusters(clusters []domainecs.Cluster) {
	sort.SliceStable(clusters, func(i, j int) bool {
		ia, ja := strings.EqualFold(clusters[i].Status, "ACTIVE"), strings.EqualFold(clusters[j].Status, "ACTIVE")
		if ia != ja {
			return ia
		}
		return strings.ToLower(clusters[i].Name) < strings.ToLower(clusters[j].Name)
	})
}

func SortServices(services []domainecs.Service) {
	sort.SliceStable(services, func(i, j int) bool {
		ia, ja := strings.EqualFold(services[i].Status, "ACTIVE"), strings.EqualFold(services[j].Status, "ACTIVE")
		if ia != ja {
			return ia
		}
		return strings.ToLower(services[i].Name) < strings.ToLower(services[j].Name)
	})
}

func SortTasks(tasks []domainecs.Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		ir, jr := strings.EqualFold(tasks[i].LastStatus, "RUNNING"), strings.EqualFold(tasks[j].LastStatus, "RUNNING")
		if ir != jr {
			return !ir
		}
		if !tasks[i].CreatedAt.Equal(tasks[j].CreatedAt) {
			return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
		}
		return tasks[i].ID < tasks[j].ID
	})
}
