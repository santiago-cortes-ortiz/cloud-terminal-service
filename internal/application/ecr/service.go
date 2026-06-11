package ecr

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	domainecr "aws-terminal/internal/domain/ecr"
)

var (
	repositoryNamePattern = regexp.MustCompile(`^(?:[a-z0-9]+(?:(?:[._-][a-z0-9]+)+)?)(?:/(?:[a-z0-9]+(?:(?:[._-][a-z0-9]+)+)?))*$`)
	tagPattern            = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`)
)

type Service struct {
	repositories RepositoryAPI
	docker       DockerAPI
}

func NewService(repositories RepositoryAPI, docker DockerAPI) *Service {
	return &Service{repositories: repositories, docker: docker}
}

type BuildPushPlanInput struct {
	Profile        string
	Region         string
	RepositoryName string
	RepositoryURI  string
	SourceImage    string
	DestinationTag string
}

func (s *Service) ListRepositories(ctx context.Context, profileName, region string) ([]domainecr.Repository, error) {
	profileName, region, err := validateProfileRegion(profileName, region)
	if err != nil {
		return nil, err
	}
	repositories, err := s.repositories.ListRepositories(ctx, profileName, region)
	if err != nil {
		return nil, err
	}
	sortRepositories(repositories)
	return repositories, nil
}

func (s *Service) SearchRepositories(ctx context.Context, profileName, region, query string) ([]domainecr.Repository, error) {
	repositories, err := s.ListRepositories(ctx, profileName, region)
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return repositories, nil
	}
	filtered := make([]domainecr.Repository, 0, len(repositories))
	for _, repository := range repositories {
		if strings.Contains(strings.ToLower(repository.Name), query) || strings.Contains(strings.ToLower(repository.URI), query) {
			filtered = append(filtered, repository)
		}
	}
	return filtered, nil
}

func (s *Service) CreateRepository(ctx context.Context, input domainecr.CreateRepositoryInput) (domainecr.Repository, error) {
	profileName, region, err := validateProfileRegion(input.Profile, input.Region)
	if err != nil {
		return domainecr.Repository{}, err
	}
	name, err := normalizeRepositoryName(input.Name)
	if err != nil {
		return domainecr.Repository{}, err
	}
	return s.repositories.CreateRepository(ctx, domainecr.CreateRepositoryInput{Profile: profileName, Region: region, Name: name})
}

func (s *Service) ListImages(ctx context.Context, profileName, region, repositoryName string) ([]domainecr.RepositoryImage, error) {
	profileName, region, err := validateProfileRegion(profileName, region)
	if err != nil {
		return nil, err
	}
	repositoryName, err = normalizeRepositoryName(repositoryName)
	if err != nil {
		return nil, err
	}
	images, err := s.repositories.ListImages(ctx, profileName, region, repositoryName)
	if err != nil {
		return nil, err
	}
	sort.Slice(images, func(i, j int) bool {
		if !images[i].PushedAt.Equal(images[j].PushedAt) {
			return images[i].PushedAt.After(images[j].PushedAt)
		}
		return images[i].Digest < images[j].Digest
	})
	return images, nil
}

func (s *Service) ListLocalImages(ctx context.Context) ([]domainecr.LocalImage, error) {
	images, err := s.docker.ListLocalImages(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(images, func(i, j int) bool {
		if images[i].Reference == images[j].Reference {
			return images[i].ID < images[j].ID
		}
		return images[i].Reference < images[j].Reference
	})
	return images, nil
}

func (s *Service) BuildPushPlan(input BuildPushPlanInput) (domainecr.PushPlan, error) {
	profileName, region, err := validateProfileRegion(input.Profile, input.Region)
	if err != nil {
		return domainecr.PushPlan{}, err
	}
	repositoryName, err := normalizeRepositoryName(input.RepositoryName)
	if err != nil {
		return domainecr.PushPlan{}, err
	}
	repositoryURI := strings.TrimSpace(input.RepositoryURI)
	if repositoryURI == "" {
		return domainecr.PushPlan{}, fmt.Errorf("repository URI is required")
	}
	sourceImage := strings.TrimSpace(input.SourceImage)
	if sourceImage == "" {
		return domainecr.PushPlan{}, fmt.Errorf("source image is required")
	}
	destinationTag := strings.TrimSpace(input.DestinationTag)
	if destinationTag == "" {
		destinationTag = defaultTagFromImage(sourceImage)
	}
	if !tagPattern.MatchString(destinationTag) {
		return domainecr.PushPlan{}, fmt.Errorf("destination tag must match Docker tag syntax")
	}
	return domainecr.PushPlan{
		Profile:          profileName,
		Region:           region,
		RepositoryName:   repositoryName,
		RepositoryURI:    repositoryURI,
		SourceImage:      sourceImage,
		DestinationTag:   destinationTag,
		DestinationImage: strings.TrimRight(repositoryURI, ":") + ":" + destinationTag,
	}, nil
}

func (s *Service) ExecutePush(ctx context.Context, plan domainecr.PushPlan, progress chan<- domainecr.PushProgress) (domainecr.PushResult, error) {
	if _, err := s.BuildPushPlan(BuildPushPlanInput{Profile: plan.Profile, Region: plan.Region, RepositoryName: plan.RepositoryName, RepositoryURI: plan.RepositoryURI, SourceImage: plan.SourceImage, DestinationTag: plan.DestinationTag}); err != nil {
		return domainecr.PushResult{}, err
	}
	auth, err := s.repositories.GetAuthorizationToken(ctx, plan.Profile, plan.Region)
	if err != nil {
		return domainecr.PushResult{}, err
	}
	if err := s.docker.Login(ctx, auth); err != nil {
		return domainecr.PushResult{}, err
	}
	if err := s.docker.TagImage(ctx, plan.SourceImage, plan.DestinationImage); err != nil {
		return domainecr.PushResult{}, err
	}
	digest, err := s.docker.PushImage(ctx, plan.DestinationImage, progress)
	if err != nil {
		return domainecr.PushResult{}, err
	}
	return domainecr.PushResult{SourceImage: plan.SourceImage, DestinationImage: plan.DestinationImage, RepositoryName: plan.RepositoryName, Tag: plan.DestinationTag, Digest: digest, CompletedAt: time.Now()}, nil
}

func validateProfileRegion(profileName, region string) (string, string, error) {
	profileName = strings.TrimSpace(profileName)
	region = strings.TrimSpace(region)
	if profileName == "" {
		return "", "", fmt.Errorf("profile name is required")
	}
	if region == "" {
		return "", "", fmt.Errorf("region is required")
	}
	return profileName, region, nil
}

func normalizeRepositoryName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("repository name is required")
	}
	if len(name) < 2 || len(name) > 256 || !repositoryNamePattern.MatchString(name) {
		return "", fmt.Errorf("repository name must be 2-256 lowercase characters and may include /, _, ., or - separators")
	}
	return name, nil
}

func sortRepositories(repositories []domainecr.Repository) {
	sort.Slice(repositories, func(i, j int) bool { return repositories[i].Name < repositories[j].Name })
}

func defaultTagFromImage(image string) string {
	image = strings.TrimSpace(image)
	lastSlash := strings.LastIndex(image, "/")
	lastColon := strings.LastIndex(image, ":")
	if lastColon > lastSlash && lastColon < len(image)-1 {
		return image[lastColon+1:]
	}
	return "latest"
}
