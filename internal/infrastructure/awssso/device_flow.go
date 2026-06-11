package awssso

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssdkconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/ssocreds"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	ssooidctypes "github.com/aws/aws-sdk-go-v2/service/ssooidc/types"

	domainauth "aws-terminal/internal/domain/auth"
	domainprofile "aws-terminal/internal/domain/profile"
)

const (
	deviceCodeGrantType = "urn:ietf:params:oauth:grant-type:device_code"
	refreshTokenGrant   = "refresh_token"
)

type OIDCDeviceFlowAuthenticator struct{}

func NewOIDCDeviceFlowAuthenticator() OIDCDeviceFlowAuthenticator {
	return OIDCDeviceFlowAuthenticator{}
}

func (OIDCDeviceFlowAuthenticator) HasReusableSession(ctx context.Context, profile domainprofile.Profile) (bool, error) {
	if !profile.UsesSSO() || profile.SSO == nil {
		return false, fmt.Errorf("profile %q is not configured for AWS SSO", profile.Name)
	}
	if strings.TrimSpace(profile.SSO.Region) == "" {
		return false, fmt.Errorf("profile %q is missing sso_region configuration", profile.Name)
	}

	cacheFilepath, err := ssocreds.StandardCachedTokenFilepath(profile.SSO.CacheKey())
	if err != nil {
		return false, fmt.Errorf("resolve AWS SSO cache path: %w", err)
	}

	client, err := newOIDCClient(ctx, profile.SSO.Region)
	if err != nil {
		return false, err
	}

	return reusableSessionFromCache(ctx, client, cacheFilepath, time.Now().UTC())
}

func reusableSessionFromCache(ctx context.Context, client ssocreds.CreateTokenAPIClient, cacheFilepath string, now time.Time) (bool, error) {
	token, err := ssocreds.NewSSOTokenProvider(client, cacheFilepath).RetrieveBearerToken(ctx)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return false, ctxErr
		}
		return false, nil
	}
	if strings.TrimSpace(token.Value) == "" {
		return false, nil
	}
	if token.CanExpire && !token.Expires.After(now) {
		return false, nil
	}

	return true, nil
}

func (OIDCDeviceFlowAuthenticator) Start(ctx context.Context, profile domainprofile.Profile) (domainauth.PendingFlow, error) {
	if !profile.UsesSSO() || profile.SSO == nil {
		return domainauth.PendingFlow{}, fmt.Errorf("profile %q is not configured for AWS SSO", profile.Name)
	}
	if strings.TrimSpace(profile.SSO.Region) == "" {
		return domainauth.PendingFlow{}, fmt.Errorf("profile %q is missing sso_region configuration", profile.Name)
	}
	if strings.TrimSpace(profile.SSO.StartURL) == "" {
		return domainauth.PendingFlow{}, fmt.Errorf("profile %q is missing sso_start_url configuration", profile.Name)
	}

	cacheFilepath, err := ssocreds.StandardCachedTokenFilepath(profile.SSO.CacheKey())
	if err != nil {
		return domainauth.PendingFlow{}, fmt.Errorf("resolve AWS SSO cache path: %w", err)
	}

	client, err := newOIDCClient(ctx, profile.SSO.Region)
	if err != nil {
		return domainauth.PendingFlow{}, err
	}

	scopes := profile.SSO.ScopesOrDefault()
	registration, err := cachedOrRegisteredClient(ctx, client, profile, cacheFilepath, scopes)
	if err != nil {
		return domainauth.PendingFlow{}, err
	}

	deviceAuth, err := client.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     aws.String(registration.ClientID),
		ClientSecret: aws.String(registration.ClientSecret),
		StartUrl:     aws.String(profile.SSO.StartURL),
	})
	if err != nil {
		return domainauth.PendingFlow{}, fmt.Errorf("start AWS SSO device authorization: %w", err)
	}

	verificationURL := aws.ToString(deviceAuth.VerificationUri)
	completeURL := aws.ToString(deviceAuth.VerificationUriComplete)
	browserTarget := completeURL
	if browserTarget == "" {
		browserTarget = verificationURL
	}
	browserOpened, browserOpenError := openBrowser(browserTarget)

	pollInterval := time.Duration(deviceAuth.Interval) * time.Second
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}

	prompt := domainauth.Prompt{
		SessionID:               newSessionID(),
		ProfileName:             profile.Name,
		VerificationURI:         verificationURL,
		VerificationURIComplete: completeURL,
		UserCode:                aws.ToString(deviceAuth.UserCode),
		ExpiresAt:               time.Now().UTC().Add(time.Duration(deviceAuth.ExpiresIn) * time.Second),
		PollInterval:            pollInterval,
		BrowserOpened:           browserOpened,
		BrowserOpenError:        browserOpenError,
	}

	return domainauth.PendingFlow{
		Prompt:                  prompt,
		CacheFilepath:           cacheFilepath,
		ClientID:                registration.ClientID,
		ClientSecret:            registration.ClientSecret,
		ClientSecretExpiresAt:   registration.ExpiresAt,
		DeviceCode:              aws.ToString(deviceAuth.DeviceCode),
		StartURL:                profile.SSO.StartURL,
		Region:                  profile.SSO.Region,
		RegistrationScopes:      scopes,
		RecommendedPollInterval: pollInterval,
	}, nil
}

type cachedClientRegistration struct {
	ClientID     string
	ClientSecret string
	ExpiresAt    time.Time
}

func cachedOrRegisteredClient(ctx context.Context, client *ssooidc.Client, profile domainprofile.Profile, tokenCacheFilepath string, scopes []string) (cachedClientRegistration, error) {
	cachePath := registrationCacheFilepath(tokenCacheFilepath, profile.SSO.Region, profile.SSO.StartURL, scopes)
	if registration, ok := readCachedClientRegistration(cachePath, time.Now().UTC()); ok {
		return registration, nil
	}

	output, err := client.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String("aws-terminal-" + profile.Name),
		ClientType: aws.String("public"),
		GrantTypes: []string{deviceCodeGrantType, refreshTokenGrant},
		Scopes:     scopes,
	})
	if err != nil {
		return cachedClientRegistration{}, fmt.Errorf("register AWS SSO OIDC client: %w", err)
	}

	registration := cachedClientRegistration{
		ClientID:     aws.ToString(output.ClientId),
		ClientSecret: aws.ToString(output.ClientSecret),
		ExpiresAt:    time.Unix(output.ClientSecretExpiresAt, 0).UTC(),
	}
	if err := writeCachedClientRegistration(cachePath, registration); err != nil {
		return cachedClientRegistration{}, err
	}

	return registration, nil
}

func registrationCacheFilepath(tokenCacheFilepath, region, startURL string, scopes []string) string {
	return filepath.Join(filepath.Dir(tokenCacheFilepath), "aws-terminal-registration-"+registrationCacheKey(region, startURL, scopes)+".json")
}

func registrationCacheKey(region, startURL string, scopes []string) string {
	scopeCopy := append([]string(nil), scopes...)
	for i := range scopeCopy {
		scopeCopy[i] = strings.TrimSpace(scopeCopy[i])
	}
	sort.Strings(scopeCopy)

	hash := sha1.Sum([]byte(strings.TrimSpace(region) + "\x00" + strings.TrimSpace(startURL) + "\x00" + strings.Join(scopeCopy, "\x00")))
	return hex.EncodeToString(hash[:])
}

func readCachedClientRegistration(path string, now time.Time) (cachedClientRegistration, bool) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return cachedClientRegistration{}, false
	}

	var record struct {
		ClientID     string `json:"clientId,omitempty"`
		ClientSecret string `json:"clientSecret,omitempty"`
		ExpiresAt    string `json:"expiresAt,omitempty"`
	}
	if err := json.Unmarshal(payload, &record); err != nil {
		return cachedClientRegistration{}, false
	}

	expiresAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(record.ExpiresAt))
	if err != nil {
		return cachedClientRegistration{}, false
	}

	registration := cachedClientRegistration{
		ClientID:     strings.TrimSpace(record.ClientID),
		ClientSecret: strings.TrimSpace(record.ClientSecret),
		ExpiresAt:    expiresAt.UTC(),
	}
	if !registration.validAt(now) {
		return cachedClientRegistration{}, false
	}

	return registration, true
}

func writeCachedClientRegistration(path string, registration cachedClientRegistration) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create AWS SSO registration cache directory: %w", err)
	}

	payload, err := json.MarshalIndent(struct {
		ClientID     string `json:"clientId,omitempty"`
		ClientSecret string `json:"clientSecret,omitempty"`
		ExpiresAt    string `json:"expiresAt,omitempty"`
	}{
		ClientID:     registration.ClientID,
		ClientSecret: registration.ClientSecret,
		ExpiresAt:    formatTimestamp(registration.ExpiresAt),
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal AWS SSO registration cache: %w", err)
	}
	payload = append(payload, '\n')

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o600); err != nil {
		return fmt.Errorf("write AWS SSO registration cache: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace AWS SSO registration cache: %w", err)
	}

	return nil
}

func (r cachedClientRegistration) validAt(now time.Time) bool {
	return strings.TrimSpace(r.ClientID) != "" &&
		strings.TrimSpace(r.ClientSecret) != "" &&
		r.ExpiresAt.After(now.Add(5*time.Minute))
}

func (OIDCDeviceFlowAuthenticator) Poll(ctx context.Context, flow *domainauth.PendingFlow) (domainauth.PollResult, error) {
	client, err := newOIDCClient(ctx, flow.Region)
	if err != nil {
		return domainauth.PollResult{}, err
	}

	result, err := client.CreateToken(ctx, &ssooidc.CreateTokenInput{
		ClientId:     aws.String(flow.ClientID),
		ClientSecret: aws.String(flow.ClientSecret),
		DeviceCode:   aws.String(flow.DeviceCode),
		GrantType:    aws.String(deviceCodeGrantType),
	})
	if err != nil {
		var authorizationPending *ssooidctypes.AuthorizationPendingException
		if errors.As(err, &authorizationPending) {
			return domainauth.PollResult{NextPollInterval: flow.RecommendedPollInterval}, nil
		}

		var slowDown *ssooidctypes.SlowDownException
		if errors.As(err, &slowDown) {
			flow.RecommendedPollInterval += 5 * time.Second
			return domainauth.PollResult{NextPollInterval: flow.RecommendedPollInterval}, nil
		}

		var expiredToken *ssooidctypes.ExpiredTokenException
		if errors.As(err, &expiredToken) {
			return domainauth.PollResult{}, fmt.Errorf("device authorization expired before completion")
		}

		var accessDenied *ssooidctypes.AccessDeniedException
		if errors.As(err, &accessDenied) {
			return domainauth.PollResult{}, fmt.Errorf("device authorization was denied")
		}

		var invalidGrant *ssooidctypes.InvalidGrantException
		if errors.As(err, &invalidGrant) {
			return domainauth.PollResult{}, fmt.Errorf("device authorization is no longer valid")
		}

		return domainauth.PollResult{}, fmt.Errorf("poll AWS SSO device authorization: %w", err)
	}

	expiresAt := time.Now().UTC().Add(time.Duration(result.ExpiresIn) * time.Second)
	record := cachedTokenRecord{
		StartURL:              flow.StartURL,
		Region:                flow.Region,
		AccessToken:           aws.ToString(result.AccessToken),
		ExpiresAt:             expiresAt,
		RefreshToken:          aws.ToString(result.RefreshToken),
		ClientID:              flow.ClientID,
		ClientSecret:          flow.ClientSecret,
		RegistrationExpiresAt: flow.ClientSecretExpiresAt,
	}
	if err := writeCachedToken(flow.CacheFilepath, record); err != nil {
		return domainauth.PollResult{}, err
	}

	return domainauth.PollResult{
		Done:   true,
		Output: "Native AWS SSO device authorization completed successfully.",
	}, nil
}

func newOIDCClient(ctx context.Context, region string) (*ssooidc.Client, error) {
	cfg, err := awssdkconfig.LoadDefaultConfig(ctx,
		awssdkconfig.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("load AWS SDK config for SSO OIDC region %q: %w", region, err)
	}

	client := ssooidc.NewFromConfig(cfg)
	return client, nil
}

type cachedTokenRecord struct {
	StartURL              string    `json:"startUrl,omitempty"`
	Region                string    `json:"region,omitempty"`
	AccessToken           string    `json:"accessToken,omitempty"`
	ExpiresAt             time.Time `json:"expiresAt,omitempty"`
	RefreshToken          string    `json:"refreshToken,omitempty"`
	ClientID              string    `json:"clientId,omitempty"`
	ClientSecret          string    `json:"clientSecret,omitempty"`
	RegistrationExpiresAt time.Time `json:"registrationExpiresAt,omitempty"`
}

func writeCachedToken(path string, record cachedTokenRecord) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create AWS SSO cache directory: %w", err)
	}

	payload, err := json.MarshalIndent(struct {
		StartURL              string `json:"startUrl,omitempty"`
		Region                string `json:"region,omitempty"`
		AccessToken           string `json:"accessToken,omitempty"`
		ExpiresAt             string `json:"expiresAt,omitempty"`
		RefreshToken          string `json:"refreshToken,omitempty"`
		ClientID              string `json:"clientId,omitempty"`
		ClientSecret          string `json:"clientSecret,omitempty"`
		RegistrationExpiresAt string `json:"registrationExpiresAt,omitempty"`
	}{
		StartURL:              record.StartURL,
		Region:                record.Region,
		AccessToken:           record.AccessToken,
		ExpiresAt:             formatTimestamp(record.ExpiresAt),
		RefreshToken:          record.RefreshToken,
		ClientID:              record.ClientID,
		ClientSecret:          record.ClientSecret,
		RegistrationExpiresAt: formatTimestamp(record.RegistrationExpiresAt),
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal AWS SSO cached token: %w", err)
	}
	payload = append(payload, '\n')

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o600); err != nil {
		return fmt.Errorf("write AWS SSO cached token: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace AWS SSO cached token: %w", err)
	}

	return nil
}

func formatTimestamp(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.UTC().Format(time.RFC3339Nano)
}

func openBrowser(target string) (bool, string) {
	if strings.TrimSpace(target) == "" {
		return false, ""
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	case "linux":
		cmd = exec.Command("xdg-open", target)
	default:
		return false, fmt.Sprintf("automatic browser opening is not supported on %s", runtime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return false, err.Error()
	}

	return true, ""
}

func newSessionID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("auth-%d", time.Now().UnixNano())
	}

	return hex.EncodeToString(bytes)
}
