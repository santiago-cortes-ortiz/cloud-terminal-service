package ecs

import (
	"context"

	domainecs "aws-terminal/internal/domain/ecs"
)

type API interface {
	ListClusters(ctx context.Context, profileName, region string) ([]domainecs.Cluster, error)
	ListServices(ctx context.Context, profileName, region, clusterARN string) ([]domainecs.Service, error)
	ListTasks(ctx context.Context, profileName, region, clusterARN string) ([]domainecs.Task, error)
}
