package awsecr

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsecrsdk "github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"

	appsecr "aws-terminal/internal/application/ecr"
	domainecr "aws-terminal/internal/domain/ecr"
	"aws-terminal/internal/infrastructure/awsclients"
)

type Service struct {
	clients *awsclients.Factory
}

func NewService() *Service { return NewServiceWithFactory(awsclients.Default()) }

func NewServiceWithFactory(clients *awsclients.Factory) *Service {
	if clients == nil {
		clients = awsclients.Default()
	}
	return &Service{clients: clients}
}

func (s *Service) ListRepositories(ctx context.Context, profileName, region string) ([]domainecr.Repository, error) {
	ctx, cancel := awsclients.WithTimeout(ctx, s.clients.OperationTimeout())
	defer cancel()
	client, err := s.client(ctx, profileName, region)
	if err != nil {
		return nil, err
	}
	paginator := awsecrsdk.NewDescribeRepositoriesPaginator(client, &awsecrsdk.DescribeRepositoriesInput{})
	repositories := make([]domainecr.Repository, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, repository := range page.Repositories {
			repositories = append(repositories, repositoryFromSDK(repository, region))
		}
	}
	sort.Slice(repositories, func(i, j int) bool { return repositories[i].Name < repositories[j].Name })
	return repositories, nil
}

func (s *Service) CreateRepository(ctx context.Context, input domainecr.CreateRepositoryInput) (domainecr.Repository, error) {
	ctx, cancel := awsclients.WithTimeout(ctx, s.clients.OperationTimeout())
	defer cancel()
	client, err := s.client(ctx, input.Profile, input.Region)
	if err != nil {
		return domainecr.Repository{}, err
	}
	out, err := client.CreateRepository(ctx, &awsecrsdk.CreateRepositoryInput{
		RepositoryName:             aws.String(strings.TrimSpace(input.Name)),
		ImageScanningConfiguration: &ecrtypes.ImageScanningConfiguration{ScanOnPush: false},
		ImageTagMutability:         ecrtypes.ImageTagMutabilityMutable,
	})
	if err != nil {
		return domainecr.Repository{}, err
	}
	if out.Repository == nil {
		return domainecr.Repository{}, fmt.Errorf("ECR returned no repository")
	}
	return repositoryFromSDK(*out.Repository, input.Region), nil
}

func (s *Service) ListImages(ctx context.Context, profileName, region, repositoryName string) ([]domainecr.RepositoryImage, error) {
	ctx, cancel := awsclients.WithTimeout(ctx, s.clients.OperationTimeout())
	defer cancel()
	client, err := s.client(ctx, profileName, region)
	if err != nil {
		return nil, err
	}
	paginator := awsecrsdk.NewDescribeImagesPaginator(client, &awsecrsdk.DescribeImagesInput{RepositoryName: aws.String(strings.TrimSpace(repositoryName))})
	images := make([]domainecr.RepositoryImage, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, detail := range page.ImageDetails {
			images = append(images, imageFromSDK(detail, repositoryName))
		}
	}
	sort.Slice(images, func(i, j int) bool {
		if !images[i].PushedAt.Equal(images[j].PushedAt) {
			return images[i].PushedAt.After(images[j].PushedAt)
		}
		return images[i].Digest < images[j].Digest
	})
	return images, nil
}

func (s *Service) GetAuthorizationToken(ctx context.Context, profileName, region string) (domainecr.AuthorizationToken, error) {
	ctx, cancel := awsclients.WithTimeout(ctx, s.clients.OperationTimeout())
	defer cancel()
	client, err := s.client(ctx, profileName, region)
	if err != nil {
		return domainecr.AuthorizationToken{}, err
	}
	out, err := client.GetAuthorizationToken(ctx, &awsecrsdk.GetAuthorizationTokenInput{})
	if err != nil {
		return domainecr.AuthorizationToken{}, err
	}
	if len(out.AuthorizationData) == 0 {
		return domainecr.AuthorizationToken{}, fmt.Errorf("ECR returned no authorization data")
	}
	data := out.AuthorizationData[0]
	decoded, err := base64.StdEncoding.DecodeString(aws.ToString(data.AuthorizationToken))
	if err != nil {
		return domainecr.AuthorizationToken{}, fmt.Errorf("decode ECR authorization token: %w", err)
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return domainecr.AuthorizationToken{}, fmt.Errorf("ECR authorization token has unexpected format")
	}
	expiresAt := aws.ToTime(data.ExpiresAt)
	return domainecr.AuthorizationToken{Username: parts[0], Password: parts[1], ProxyEndpoint: strings.TrimSpace(aws.ToString(data.ProxyEndpoint)), ExpiresAt: expiresAt}, nil
}

var _ appsecr.RepositoryAPI = (*Service)(nil)

func (s *Service) client(ctx context.Context, profileName, region string) (*awsecrsdk.Client, error) {
	client, err := s.clients.ECR(ctx, profileName, region)
	if err != nil {
		return nil, fmt.Errorf("load ECR client: %w", err)
	}
	return client, nil
}

func repositoryFromSDK(repository ecrtypes.Repository, fallbackRegion string) domainecr.Repository {
	createdAt := aws.ToTime(repository.CreatedAt)
	scanOnPush := false
	if repository.ImageScanningConfiguration != nil {
		scanOnPush = repository.ImageScanningConfiguration.ScanOnPush
	}
	return domainecr.Repository{
		Name:        aws.ToString(repository.RepositoryName),
		URI:         aws.ToString(repository.RepositoryUri),
		RegistryID:  aws.ToString(repository.RegistryId),
		Region:      strings.TrimSpace(fallbackRegion),
		CreatedAt:   createdAt,
		ScanOnPush:  scanOnPush,
		MutableTags: repository.ImageTagMutability == ecrtypes.ImageTagMutabilityMutable,
	}
}

func imageFromSDK(detail ecrtypes.ImageDetail, repositoryName string) domainecr.RepositoryImage {
	return domainecr.RepositoryImage{
		RepositoryName:     strings.TrimSpace(repositoryName),
		Digest:             aws.ToString(detail.ImageDigest),
		Tags:               append([]string(nil), detail.ImageTags...),
		SizeBytes:          aws.ToInt64(detail.ImageSizeInBytes),
		PushedAt:           aws.ToTime(detail.ImagePushedAt),
		LastRecordedPullAt: aws.ToTime(detail.LastRecordedPullTime),
	}
}
