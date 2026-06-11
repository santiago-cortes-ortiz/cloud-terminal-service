package ecr

import (
	"context"

	domainecr "aws-terminal/internal/domain/ecr"
)

type RepositoryAPI interface {
	ListRepositories(ctx context.Context, profileName, region string) ([]domainecr.Repository, error)
	CreateRepository(ctx context.Context, input domainecr.CreateRepositoryInput) (domainecr.Repository, error)
	ListImages(ctx context.Context, profileName, region, repositoryName string) ([]domainecr.RepositoryImage, error)
	GetAuthorizationToken(ctx context.Context, profileName, region string) (domainecr.AuthorizationToken, error)
}

type DockerAPI interface {
	ListLocalImages(ctx context.Context) ([]domainecr.LocalImage, error)
	Login(ctx context.Context, auth domainecr.AuthorizationToken) error
	TagImage(ctx context.Context, sourceImage, destinationImage string) error
	PushImage(ctx context.Context, destinationImage string, progress chan<- domainecr.PushProgress) (string, error)
}
