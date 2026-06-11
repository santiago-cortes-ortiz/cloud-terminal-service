package cloudfront

import (
	"context"
	"fmt"
	"sort"
	"strings"

	domaincloudfront "aws-terminal/internal/domain/cloudfront"
)

type Service struct {
	api API
}

func NewService(api API) *Service {
	return &Service{api: api}
}

func (s *Service) ListDistributions(ctx context.Context, profileName, region string) ([]domaincloudfront.Distribution, error) {
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		return nil, fmt.Errorf("profile name is required")
	}

	distributions, err := s.api.ListDistributions(ctx, profileName, strings.TrimSpace(region))
	if err != nil {
		return nil, err
	}

	sort.Slice(distributions, func(i, j int) bool {
		return distributions[i].ID < distributions[j].ID
	})
	return distributions, nil
}

func (s *Service) CreateInvalidation(ctx context.Context, input CreateInvalidationInput) (domaincloudfront.Invalidation, error) {
	input.Profile = strings.TrimSpace(input.Profile)
	if input.Profile == "" {
		return domaincloudfront.Invalidation{}, fmt.Errorf("profile name is required")
	}
	input.DistributionID = strings.TrimSpace(input.DistributionID)
	if input.DistributionID == "" {
		return domaincloudfront.Invalidation{}, fmt.Errorf("distribution ID is required")
	}

	paths := normalizePaths(input.Paths)
	if len(paths) == 0 {
		return domaincloudfront.Invalidation{}, fmt.Errorf("at least one invalidation path is required")
	}
	input.Paths = paths
	input.Region = strings.TrimSpace(input.Region)

	return s.api.CreateInvalidation(ctx, input)
}

func (s *Service) GetInvalidation(ctx context.Context, profileName, region, distributionID, invalidationID string) (domaincloudfront.Invalidation, error) {
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		return domaincloudfront.Invalidation{}, fmt.Errorf("profile name is required")
	}
	distributionID = strings.TrimSpace(distributionID)
	if distributionID == "" {
		return domaincloudfront.Invalidation{}, fmt.Errorf("distribution ID is required")
	}
	invalidationID = strings.TrimSpace(invalidationID)
	if invalidationID == "" {
		return domaincloudfront.Invalidation{}, fmt.Errorf("invalidation ID is required")
	}

	return s.api.GetInvalidation(ctx, profileName, strings.TrimSpace(region), distributionID, invalidationID)
}

func normalizePaths(paths []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		result = append(result, path)
	}
	return result
}
