package awssso

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
)

type mockCreateTokenClient struct {
	called bool
	input  *ssooidc.CreateTokenInput
	output *ssooidc.CreateTokenOutput
	err    error
}

func (m *mockCreateTokenClient) CreateToken(ctx context.Context, input *ssooidc.CreateTokenInput, optFns ...func(*ssooidc.Options)) (*ssooidc.CreateTokenOutput, error) {
	m.called = true
	m.input = input
	if m.err != nil {
		return nil, m.err
	}
	return m.output, nil
}

func TestReusableSessionFromCacheReturnsTrueForUnexpiredToken(t *testing.T) {
	now := time.Now().UTC()
	path := writeTestTokenCache(t, map[string]string{
		"accessToken": "token",
		"expiresAt":   now.Add(time.Hour).Format(time.RFC3339),
	})
	client := &mockCreateTokenClient{}

	reusable, err := reusableSessionFromCache(context.Background(), client, path, now)
	if err != nil {
		t.Fatalf("reusableSessionFromCache() error = %v", err)
	}
	if !reusable {
		t.Fatal("expected reusable token")
	}
	if client.called {
		t.Fatal("did not expect refresh for unexpired token")
	}
}

func TestReusableSessionFromCacheRefreshesExpiredToken(t *testing.T) {
	now := time.Now().UTC()
	path := writeTestTokenCache(t, map[string]string{
		"accessToken":  "old-token",
		"expiresAt":    now.Add(-time.Hour).Format(time.RFC3339),
		"refreshToken": "refresh-token",
		"clientId":     "client-id",
		"clientSecret": "client-secret",
	})
	client := &mockCreateTokenClient{output: &ssooidc.CreateTokenOutput{
		AccessToken:  aws.String("new-token"),
		RefreshToken: aws.String("new-refresh-token"),
		ExpiresIn:    3600,
	}}

	reusable, err := reusableSessionFromCache(context.Background(), client, path, now)
	if err != nil {
		t.Fatalf("reusableSessionFromCache() error = %v", err)
	}
	if !reusable {
		t.Fatal("expected refreshed token to be reusable")
	}
	if !client.called {
		t.Fatal("expected expired token to be refreshed")
	}
	if got := aws.ToString(client.input.GrantType); got != refreshTokenGrant {
		t.Fatalf("grant type = %q, want %q", got, refreshTokenGrant)
	}
}

func TestReusableSessionFromCacheReturnsFalseForMissingOrUnrefreshableCache(t *testing.T) {
	now := time.Now().UTC()
	missing := filepath.Join(t.TempDir(), "missing.json")
	reusable, err := reusableSessionFromCache(context.Background(), &mockCreateTokenClient{}, missing, now)
	if err != nil {
		t.Fatalf("missing cache error = %v", err)
	}
	if reusable {
		t.Fatal("expected missing cache to be not reusable")
	}

	path := writeTestTokenCache(t, map[string]string{
		"accessToken": "old-token",
		"expiresAt":   now.Add(-time.Hour).Format(time.RFC3339),
	})
	reusable, err = reusableSessionFromCache(context.Background(), &mockCreateTokenClient{}, path, now)
	if err != nil {
		t.Fatalf("unrefreshable cache error = %v", err)
	}
	if reusable {
		t.Fatal("expected expired unrefreshable cache to be not reusable")
	}
}

func writeTestTokenCache(t *testing.T, fields map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "token.json")
	payload, err := json.Marshal(fields)
	if err != nil {
		t.Fatalf("marshal token: %v", err)
	}
	if err := writeFileForTest(path, payload); err != nil {
		t.Fatalf("write token: %v", err)
	}
	return path
}

func writeFileForTest(path string, payload []byte) error {
	return os.WriteFile(path, payload, 0o600)
}
