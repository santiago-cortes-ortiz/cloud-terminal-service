package session

import (
	"context"

	domainprofile "aws-terminal/internal/domain/profile"
	domainsession "aws-terminal/internal/domain/session"
)

type ProfileRepository interface {
	List(ctx context.Context) ([]domainprofile.Profile, error)
}

type IdentityResolver interface {
	Resolve(ctx context.Context, profileName, region string) (domainsession.Session, error)
}
