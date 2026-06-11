package awsconfig

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	domainsession "aws-terminal/internal/domain/session"
	"aws-terminal/internal/infrastructure/awsclients"
)

type STSIdentityResolver struct {
	clients *awsclients.Factory
}

func NewSTSIdentityResolver() STSIdentityResolver {
	return STSIdentityResolver{clients: awsclients.Default()}
}

func NewSTSIdentityResolverWithFactory(clients *awsclients.Factory) STSIdentityResolver {
	if clients == nil {
		clients = awsclients.Default()
	}
	return STSIdentityResolver{clients: clients}
}

func (r STSIdentityResolver) Resolve(ctx context.Context, profileName, region string) (domainsession.Session, error) {
	ctx, cancel := awsclients.WithTimeout(ctx, r.clients.OperationTimeout())
	defer cancel()

	client, cfg, err := r.clients.STS(ctx, profileName, region)
	if err != nil {
		return domainsession.Session{}, err
	}

	identity, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return domainsession.Session{}, err
	}

	return domainsession.Session{
		Profile: profileName,
		Account: aws.ToString(identity.Account),
		ARN:     aws.ToString(identity.Arn),
		UserID:  aws.ToString(identity.UserId),
		Region:  cfg.Region,
	}, nil
}
