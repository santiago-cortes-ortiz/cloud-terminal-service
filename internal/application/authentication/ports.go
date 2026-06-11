package authentication

import (
	"context"

	domainauth "aws-terminal/internal/domain/auth"
	domainprofile "aws-terminal/internal/domain/profile"
)

type DeviceFlowAuthenticator interface {
	HasReusableSession(ctx context.Context, profile domainprofile.Profile) (bool, error)
	Start(ctx context.Context, profile domainprofile.Profile) (domainauth.PendingFlow, error)
	Poll(ctx context.Context, flow *domainauth.PendingFlow) (domainauth.PollResult, error)
}
