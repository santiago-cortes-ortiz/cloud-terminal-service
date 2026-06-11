package awsconfig

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"

	"aws-terminal/internal/infrastructure/awsclients"
)

func Load(ctx context.Context, profileName, region string) (aws.Config, error) {
	return awsclients.Default().Config(ctx, profileName, region)
}
