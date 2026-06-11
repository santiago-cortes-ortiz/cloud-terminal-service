package cloudfront

import (
	"context"

	domaincloudfront "aws-terminal/internal/domain/cloudfront"
)

type CreateInvalidationInput struct {
	Profile        string
	Region         string
	DistributionID string
	Paths          []string
}

type API interface {
	ListDistributions(ctx context.Context, profileName, region string) ([]domaincloudfront.Distribution, error)
	CreateInvalidation(ctx context.Context, input CreateInvalidationInput) (domaincloudfront.Invalidation, error)
	GetInvalidation(ctx context.Context, profileName, region, distributionID, invalidationID string) (domaincloudfront.Invalidation, error)
}
