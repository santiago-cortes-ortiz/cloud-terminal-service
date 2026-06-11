package profile

import "strings"

type AuthenticationMode string

const (
	AuthModeCredentials AuthenticationMode = "credentials"
	AuthModeSSO         AuthenticationMode = "sso"
)

const DefaultSSORegistrationScope = "sso:account:access"

type SSOConfiguration struct {
	SessionName        string
	StartURL           string
	Region             string
	RegistrationScopes []string
}

func (c SSOConfiguration) CacheKey() string {
	if strings.TrimSpace(c.SessionName) != "" {
		return strings.TrimSpace(c.SessionName)
	}

	return strings.TrimSpace(c.StartURL)
}

func (c SSOConfiguration) ScopesOrDefault() []string {
	if len(c.RegistrationScopes) == 0 {
		return []string{DefaultSSORegistrationScope}
	}

	return c.RegistrationScopes
}

type Profile struct {
	Name               string
	DefaultRegion      string
	AuthenticationMode AuthenticationMode
	SSO                *SSOConfiguration
}

func (p Profile) UsesSSO() bool {
	return p.AuthenticationMode == AuthModeSSO
}
