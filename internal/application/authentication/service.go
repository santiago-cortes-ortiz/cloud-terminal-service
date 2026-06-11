package authentication

import (
	"context"
	"fmt"
	"strings"
	"sync"

	domainauth "aws-terminal/internal/domain/auth"
	domainprofile "aws-terminal/internal/domain/profile"
)

type Service struct {
	deviceFlow DeviceFlowAuthenticator
	mu         sync.Mutex
	sessions   map[string]domainauth.PendingFlow
}

func NewService(deviceFlow DeviceFlowAuthenticator) *Service {
	return &Service{
		deviceFlow: deviceFlow,
		sessions:   map[string]domainauth.PendingFlow{},
	}
}

func (s *Service) HasReusableSSOSession(ctx context.Context, profile domainprofile.Profile) (bool, error) {
	if err := validateSSOProfile(profile); err != nil {
		return false, err
	}

	return s.deviceFlow.HasReusableSession(ctx, profile)
}

func (s *Service) StartSSOLogin(ctx context.Context, profile domainprofile.Profile) (domainauth.Prompt, error) {
	if err := validateSSOProfile(profile); err != nil {
		return domainauth.Prompt{}, err
	}

	flow, err := s.deviceFlow.Start(ctx, profile)
	if err != nil {
		return domainauth.Prompt{}, err
	}

	s.mu.Lock()
	s.sessions[flow.Prompt.SessionID] = flow
	s.mu.Unlock()

	return flow.Prompt, nil
}

func validateSSOProfile(profile domainprofile.Profile) error {
	if strings.TrimSpace(profile.Name) == "" {
		return fmt.Errorf("profile name is required")
	}
	if !profile.UsesSSO() {
		return fmt.Errorf("profile %q is not configured for AWS SSO", profile.Name)
	}
	if profile.SSO == nil {
		return fmt.Errorf("profile %q is not configured for AWS SSO", profile.Name)
	}
	if strings.TrimSpace(profile.SSO.Region) == "" {
		return fmt.Errorf("profile %q is missing sso_region configuration", profile.Name)
	}
	if strings.TrimSpace(profile.SSO.StartURL) == "" {
		return fmt.Errorf("profile %q is missing sso_start_url configuration", profile.Name)
	}
	return nil
}

func (s *Service) PollSSOLogin(ctx context.Context, sessionID string) (domainauth.PollResult, error) {
	s.mu.Lock()
	flow, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if !ok {
		return domainauth.PollResult{}, fmt.Errorf("sso login session %q was not found", sessionID)
	}

	result, err := s.deviceFlow.Poll(ctx, &flow)
	if err != nil {
		s.mu.Lock()
		delete(s.sessions, sessionID)
		s.mu.Unlock()
		return domainauth.PollResult{}, err
	}

	s.mu.Lock()
	if result.Done {
		delete(s.sessions, sessionID)
	} else {
		s.sessions[sessionID] = flow
	}
	s.mu.Unlock()

	return result, nil
}
