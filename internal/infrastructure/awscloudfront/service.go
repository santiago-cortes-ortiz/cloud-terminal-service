package awscloudfront

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscloudfrontsdk "github.com/aws/aws-sdk-go-v2/service/cloudfront"
	cloudfronttypes "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"

	appcloudfront "aws-terminal/internal/application/cloudfront"
	domaincloudfront "aws-terminal/internal/domain/cloudfront"
	"aws-terminal/internal/infrastructure/awsclients"
)

type Service struct {
	clients *awsclients.Factory
}

func NewService() *Service {
	return NewServiceWithFactory(awsclients.Default())
}

func NewServiceWithFactory(clients *awsclients.Factory) *Service {
	if clients == nil {
		clients = awsclients.Default()
	}
	return &Service{clients: clients}
}

func (s *Service) ListDistributions(ctx context.Context, profileName, region string) ([]domaincloudfront.Distribution, error) {
	ctx, cancel := awsclients.WithTimeout(ctx, s.clients.OperationTimeout())
	defer cancel()

	client, err := s.client(ctx, profileName, region)
	if err != nil {
		return nil, err
	}

	paginator := awscloudfrontsdk.NewListDistributionsPaginator(client, &awscloudfrontsdk.ListDistributionsInput{})
	distributions := make([]domaincloudfront.Distribution, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		if page.DistributionList == nil {
			continue
		}
		for _, item := range page.DistributionList.Items {
			distributions = append(distributions, domaincloudfront.Distribution{
				ID:         aws.ToString(item.Id),
				DomainName: aws.ToString(item.DomainName),
				Comment:    aws.ToString(item.Comment),
				Aliases:    aliases(item.Aliases),
				Enabled:    aws.ToBool(item.Enabled),
			})
		}
	}

	sort.Slice(distributions, func(i, j int) bool {
		return distributions[i].ID < distributions[j].ID
	})
	return distributions, nil
}

func (s *Service) CreateInvalidation(ctx context.Context, input appcloudfront.CreateInvalidationInput) (domaincloudfront.Invalidation, error) {
	ctx, cancel := awsclients.WithTimeout(ctx, s.clients.OperationTimeout())
	defer cancel()

	client, err := s.client(ctx, input.Profile, input.Region)
	if err != nil {
		return domaincloudfront.Invalidation{}, err
	}

	output, err := client.CreateInvalidation(ctx, createInvalidationRequest(input, fmt.Sprintf("aws-terminal-%d", time.Now().UnixNano())))
	if err != nil {
		return domaincloudfront.Invalidation{}, err
	}
	if output.Invalidation == nil {
		return domaincloudfront.Invalidation{}, fmt.Errorf("cloudfront returned no invalidation")
	}

	return invalidationFromSDK(output.Invalidation, strings.TrimSpace(input.DistributionID), input.Paths), nil
}

func (s *Service) GetInvalidation(ctx context.Context, profileName, region, distributionID, invalidationID string) (domaincloudfront.Invalidation, error) {
	ctx, cancel := awsclients.WithTimeout(ctx, s.clients.OperationTimeout())
	defer cancel()

	client, err := s.client(ctx, profileName, region)
	if err != nil {
		return domaincloudfront.Invalidation{}, err
	}

	output, err := client.GetInvalidation(ctx, getInvalidationRequest(distributionID, invalidationID))
	if err != nil {
		return domaincloudfront.Invalidation{}, err
	}
	if output.Invalidation == nil {
		return domaincloudfront.Invalidation{}, fmt.Errorf("cloudfront returned no invalidation")
	}

	return invalidationFromSDK(output.Invalidation, strings.TrimSpace(distributionID), nil), nil
}

func createInvalidationRequest(input appcloudfront.CreateInvalidationInput, callerReference string) *awscloudfrontsdk.CreateInvalidationInput {
	quantity := int32(len(input.Paths))
	return &awscloudfrontsdk.CreateInvalidationInput{
		DistributionId: aws.String(strings.TrimSpace(input.DistributionID)),
		InvalidationBatch: &cloudfronttypes.InvalidationBatch{
			CallerReference: aws.String(strings.TrimSpace(callerReference)),
			Paths: &cloudfronttypes.Paths{
				Items:    append([]string(nil), input.Paths...),
				Quantity: aws.Int32(quantity),
			},
		},
	}
}

func getInvalidationRequest(distributionID, invalidationID string) *awscloudfrontsdk.GetInvalidationInput {
	return &awscloudfrontsdk.GetInvalidationInput{
		DistributionId: aws.String(strings.TrimSpace(distributionID)),
		Id:             aws.String(strings.TrimSpace(invalidationID)),
	}
}

func (s *Service) client(ctx context.Context, profileName, region string) (*awscloudfrontsdk.Client, error) {
	client, err := s.clients.CloudFront(ctx, profileName, region)
	if err != nil {
		return nil, fmt.Errorf("load CloudFront client: %w", err)
	}
	return client, nil
}

func invalidationFromSDK(input *cloudfronttypes.Invalidation, distributionID string, fallbackPaths []string) domaincloudfront.Invalidation {
	createdAt := time.Time{}
	if input.CreateTime != nil {
		createdAt = *input.CreateTime
	}

	paths := append([]string(nil), fallbackPaths...)
	if input.InvalidationBatch != nil && input.InvalidationBatch.Paths != nil && len(input.InvalidationBatch.Paths.Items) > 0 {
		paths = append([]string(nil), input.InvalidationBatch.Paths.Items...)
	}

	return domaincloudfront.Invalidation{
		ID:             aws.ToString(input.Id),
		Status:         aws.ToString(input.Status),
		DistributionID: distributionID,
		Paths:          paths,
		CreatedAt:      createdAt,
	}
}

func aliases(input *cloudfronttypes.Aliases) []string {
	if input == nil || len(input.Items) == 0 {
		return nil
	}
	result := make([]string, 0, len(input.Items))
	for _, alias := range input.Items {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}
		result = append(result, alias)
	}
	return result
}
