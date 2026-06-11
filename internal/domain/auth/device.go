package auth

import "time"

type Prompt struct {
	SessionID               string
	ProfileName             string
	VerificationURI         string
	VerificationURIComplete string
	UserCode                string
	ExpiresAt               time.Time
	PollInterval            time.Duration
	BrowserOpened           bool
	BrowserOpenError        string
}

type PendingFlow struct {
	Prompt                  Prompt
	CacheFilepath           string
	ClientID                string
	ClientSecret            string
	ClientSecretExpiresAt   time.Time
	DeviceCode              string
	StartURL                string
	Region                  string
	RegistrationScopes      []string
	RecommendedPollInterval time.Duration
}

type PollResult struct {
	Done             bool
	Output           string
	NextPollInterval time.Duration
}
