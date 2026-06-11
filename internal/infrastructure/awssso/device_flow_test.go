package awssso

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRegistrationCacheKeyIgnoresScopeOrder(t *testing.T) {
	first := registrationCacheKey(" eu-west-1 ", " https://example.awsapps.com/start ", []string{"b", "a"})
	second := registrationCacheKey("eu-west-1", "https://example.awsapps.com/start", []string{"a", "b"})

	if first != second {
		t.Fatalf("expected cache key to ignore scope order, got %q and %q", first, second)
	}
}

func TestRegistrationCacheFilepathUsesTokenCacheDirectory(t *testing.T) {
	path := registrationCacheFilepath(filepath.Join("tmp", "cache", "token.json"), "eu-west-1", "start", []string{"sso:account:access"})
	if filepath.Dir(path) != filepath.Join("tmp", "cache") {
		t.Fatalf("expected token cache directory, got %q", filepath.Dir(path))
	}
	if !strings.HasPrefix(filepath.Base(path), "aws-terminal-registration-") {
		t.Fatalf("unexpected registration cache filename %q", filepath.Base(path))
	}
}

func TestCachedClientRegistrationValidity(t *testing.T) {
	now := time.Now().UTC()
	registration := cachedClientRegistration{
		ClientID:     "client",
		ClientSecret: "secret",
		ExpiresAt:    now.Add(10 * time.Minute),
	}
	if !registration.validAt(now) {
		t.Fatal("expected registration to be valid")
	}

	registration.ExpiresAt = now.Add(4 * time.Minute)
	if registration.validAt(now) {
		t.Fatal("expected soon-expiring registration to be invalid")
	}
}

func TestReadWriteCachedClientRegistration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registration.json")
	now := time.Now().UTC()
	want := cachedClientRegistration{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		ExpiresAt:    now.Add(time.Hour).Truncate(0),
	}

	if err := writeCachedClientRegistration(path, want); err != nil {
		t.Fatalf("writeCachedClientRegistration() error = %v", err)
	}

	got, ok := readCachedClientRegistration(path, now)
	if !ok {
		t.Fatal("expected cached registration")
	}
	if got.ClientID != want.ClientID || got.ClientSecret != want.ClientSecret || !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Fatalf("cached registration = %#v, want %#v", got, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat cache file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("cache file mode = %v, want 0600", info.Mode().Perm())
	}
}
