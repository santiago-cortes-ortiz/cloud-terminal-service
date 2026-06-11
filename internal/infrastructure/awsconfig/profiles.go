package awsconfig

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	domainprofile "aws-terminal/internal/domain/profile"
)

type SharedConfigProfileRepository struct{}

func NewSharedConfigProfileRepository() SharedConfigProfileRepository {
	return SharedConfigProfileRepository{}
}

func (SharedConfigProfileRepository) List(ctx context.Context) ([]domainprofile.Profile, error) {
	_ = ctx

	profilesByName := map[string]domainprofile.Profile{}

	configProfiles, err := loadProfilesFromConfigFile(awsConfigPath())
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		for name, profile := range configProfiles {
			profilesByName[name] = profile
		}
	}

	credentialProfiles, err := loadProfilesFromCredentialsFile(awsCredentialsPath())
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		for name, profile := range credentialProfiles {
			existing, found := profilesByName[name]
			if found {
				if existing.AuthenticationMode == "" {
					existing.AuthenticationMode = profile.AuthenticationMode
				}
				profilesByName[name] = existing
				continue
			}
			profilesByName[name] = profile
		}
	}

	profiles := make([]domainprofile.Profile, 0, len(profilesByName))
	for _, profile := range profilesByName {
		if profile.AuthenticationMode == "" {
			profile.AuthenticationMode = domainprofile.AuthModeCredentials
		}
		if profile.UsesSSO() && profile.SSO != nil {
			profile.SSO.RegistrationScopes = normalizeScopes(profile.SSO.RegistrationScopes)
		}
		profiles = append(profiles, profile)
	}

	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].Name == "default" {
			return true
		}
		if profiles[j].Name == "default" {
			return false
		}

		return profiles[i].Name < profiles[j].Name
	})

	return profiles, nil
}

func loadProfilesFromConfigFile(path string) (map[string]domainprofile.Profile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	profiles := map[string]domainprofile.Profile{}
	sessions := map[string]domainprofile.SSOConfiguration{}
	var currentProfile string
	var currentSession string
	currentSectionType := ""

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentProfile = ""
			currentSession = ""
			currentSectionType = ""

			sectionName := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			switch {
			case sectionName == "default":
				currentSectionType = "profile"
				currentProfile = "default"
				profile := profiles[currentProfile]
				profile.Name = currentProfile
				if profile.AuthenticationMode == "" {
					profile.AuthenticationMode = domainprofile.AuthModeCredentials
				}
				profiles[currentProfile] = profile
			case strings.HasPrefix(sectionName, "profile "):
				currentSectionType = "profile"
				currentProfile = strings.TrimSpace(strings.TrimPrefix(sectionName, "profile "))
				profile := profiles[currentProfile]
				profile.Name = currentProfile
				if profile.AuthenticationMode == "" {
					profile.AuthenticationMode = domainprofile.AuthModeCredentials
				}
				profiles[currentProfile] = profile
			case strings.HasPrefix(sectionName, "sso-session "):
				currentSectionType = "sso-session"
				currentSession = strings.TrimSpace(strings.TrimPrefix(sectionName, "sso-session "))
				session := sessions[currentSession]
				session.SessionName = currentSession
				sessions[currentSession] = session
			}
			continue
		}

		key, value, found := parseKeyValue(line)
		if !found {
			continue
		}

		switch currentSectionType {
		case "profile":
			profile := profiles[currentProfile]
			applyProfileKey(&profile, key, value)
			profiles[currentProfile] = profile
		case "sso-session":
			session := sessions[currentSession]
			applySSOSessionKey(&session, key, value)
			sessions[currentSession] = session
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	for name, profile := range profiles {
		if profile.SSO != nil && strings.TrimSpace(profile.SSO.SessionName) != "" {
			if session, found := sessions[profile.SSO.SessionName]; found {
				merged := *profile.SSO
				if merged.StartURL == "" {
					merged.StartURL = session.StartURL
				}
				if merged.Region == "" {
					merged.Region = session.Region
				}
				if len(merged.RegistrationScopes) == 0 {
					merged.RegistrationScopes = session.RegistrationScopes
				}
				profile.SSO = &merged
			}
		}
		profiles[name] = profile
	}

	return profiles, nil
}

func loadProfilesFromCredentialsFile(path string) (map[string]domainprofile.Profile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	profiles := map[string]domainprofile.Profile{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			profiles[name] = domainprofile.Profile{
				Name:               name,
				AuthenticationMode: domainprofile.AuthModeCredentials,
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	return profiles, nil
}

func applyProfileKey(profile *domainprofile.Profile, key, value string) {
	switch key {
	case "region":
		profile.DefaultRegion = value
	case "sso_session":
		profile.AuthenticationMode = domainprofile.AuthModeSSO
		ensureSSO(profile).SessionName = value
	case "sso_start_url":
		profile.AuthenticationMode = domainprofile.AuthModeSSO
		ensureSSO(profile).StartURL = value
	case "sso_region":
		profile.AuthenticationMode = domainprofile.AuthModeSSO
		ensureSSO(profile).Region = value
	case "sso_registration_scopes":
		profile.AuthenticationMode = domainprofile.AuthModeSSO
		ensureSSO(profile).RegistrationScopes = parseScopes(value)
	case "sso_account_id", "sso_role_name":
		profile.AuthenticationMode = domainprofile.AuthModeSSO
		ensureSSO(profile)
	}
}

func applySSOSessionKey(session *domainprofile.SSOConfiguration, key, value string) {
	switch key {
	case "sso_start_url":
		session.StartURL = value
	case "sso_region":
		session.Region = value
	case "sso_registration_scopes":
		session.RegistrationScopes = parseScopes(value)
	}
}

func ensureSSO(profile *domainprofile.Profile) *domainprofile.SSOConfiguration {
	if profile.SSO == nil {
		profile.SSO = &domainprofile.SSOConfiguration{}
	}

	return profile.SSO
}

func parseKeyValue(line string) (string, string, bool) {
	key, value, found := strings.Cut(line, "=")
	if !found {
		key, value, found = strings.Cut(line, ":")
	}
	if !found {
		return "", "", false
	}

	return strings.TrimSpace(key), strings.TrimSpace(value), true
}

func parseScopes(value string) []string {
	parts := strings.Split(value, ",")
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		scopes = append(scopes, trimmed)
	}

	return normalizeScopes(scopes)
}

func normalizeScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return []string{domainprofile.DefaultSSORegistrationScope}
	}

	unique := make([]string, 0, len(scopes))
	seen := map[string]struct{}{}
	for _, scope := range scopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed == "" {
			continue
		}
		if _, found := seen[trimmed]; found {
			continue
		}
		seen[trimmed] = struct{}{}
		unique = append(unique, trimmed)
	}
	if len(unique) == 0 {
		return []string{domainprofile.DefaultSSORegistrationScope}
	}

	return unique
}

func awsConfigPath() string {
	if path := strings.TrimSpace(os.Getenv("AWS_CONFIG_FILE")); path != "" {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".aws", "config")
	}

	return filepath.Join(home, ".aws", "config")
}

func awsCredentialsPath() string {
	if path := strings.TrimSpace(os.Getenv("AWS_SHARED_CREDENTIALS_FILE")); path != "" {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".aws", "credentials")
	}

	return filepath.Join(home, ".aws", "credentials")
}
