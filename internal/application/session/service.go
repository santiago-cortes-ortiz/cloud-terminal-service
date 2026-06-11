package session

import (
	"context"
	"fmt"
	"strings"

	domainprofile "aws-terminal/internal/domain/profile"
	domainsession "aws-terminal/internal/domain/session"
)

type Service struct {
	profiles   ProfileRepository
	identities IdentityResolver
}

func NewService(profiles ProfileRepository, identities IdentityResolver) *Service {
	return &Service{
		profiles:   profiles,
		identities: identities,
	}
}

func (s *Service) ListProfiles(ctx context.Context) ([]domainprofile.Profile, error) {
	return s.profiles.List(ctx)
}

func (s *Service) ActivateProfile(ctx context.Context, profileName, region string) (domainsession.Session, error) {
	if strings.TrimSpace(profileName) == "" {
		return domainsession.Session{}, fmt.Errorf("profile name is required")
	}

	return s.identities.Resolve(ctx, profileName, region)
}
