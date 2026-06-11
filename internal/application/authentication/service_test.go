package authentication

import (
	"context"
	"errors"
	"testing"

	domainauth "aws-terminal/internal/domain/auth"
	domainprofile "aws-terminal/internal/domain/profile"
)

type fakeDeviceFlowAuthenticator struct {
	reusable bool
	err      error
	called   bool
}

func (f *fakeDeviceFlowAuthenticator) HasReusableSession(context.Context, domainprofile.Profile) (bool, error) {
	f.called = true
	return f.reusable, f.err
}

func (f *fakeDeviceFlowAuthenticator) Start(context.Context, domainprofile.Profile) (domainauth.PendingFlow, error) {
	return domainauth.PendingFlow{}, nil
}

func (f *fakeDeviceFlowAuthenticator) Poll(context.Context, *domainauth.PendingFlow) (domainauth.PollResult, error) {
	return domainauth.PollResult{}, nil
}

func TestHasReusableSSOSessionValidatesProfile(t *testing.T) {
	service := NewService(&fakeDeviceFlowAuthenticator{})
	if _, err := service.HasReusableSSOSession(context.Background(), domainprofile.Profile{}); err == nil {
		t.Fatal("expected missing profile name error")
	}

	_, err := service.HasReusableSSOSession(context.Background(), domainprofile.Profile{
		Name:               "dev",
		AuthenticationMode: domainprofile.AuthModeCredentials,
	})
	if err == nil {
		t.Fatal("expected non-SSO profile error")
	}
}

func TestHasReusableSSOSessionDelegatesToAuthenticator(t *testing.T) {
	adapter := &fakeDeviceFlowAuthenticator{reusable: true}
	service := NewService(adapter)

	reusable, err := service.HasReusableSSOSession(context.Background(), testSSOProfile())
	if err != nil {
		t.Fatalf("HasReusableSSOSession() error = %v", err)
	}
	if !reusable || !adapter.called {
		t.Fatalf("expected reusable delegation, reusable=%v called=%v", reusable, adapter.called)
	}
}

func TestHasReusableSSOSessionReturnsAuthenticatorError(t *testing.T) {
	wantErr := errors.New("boom")
	service := NewService(&fakeDeviceFlowAuthenticator{err: wantErr})

	_, err := service.HasReusableSSOSession(context.Background(), testSSOProfile())
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

func testSSOProfile() domainprofile.Profile {
	return domainprofile.Profile{
		Name:               "dev",
		AuthenticationMode: domainprofile.AuthModeSSO,
		SSO: &domainprofile.SSOConfiguration{
			StartURL: "https://example.awsapps.com/start",
			Region:   "us-east-1",
		},
	}
}
