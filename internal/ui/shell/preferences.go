package shell

import "strings"

func (m *Model) savePreferences() {
	if m.preferenceStore == nil {
		return
	}
	_ = m.preferenceStore.Save(m.preferences)
}

func (m *Model) rememberProfile(profileName string) {
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		return
	}
	m.preferences.LastProfile = profileName
	m.savePreferences()
}

func (m *Model) rememberRegion(regionID string) {
	regionID = strings.TrimSpace(regionID)
	if regionID == "" {
		return
	}
	m.preferences.LastRegion = regionID
	m.savePreferences()
}

func (m *Model) rememberPage(pageID string) {
	pageID = strings.TrimSpace(pageID)
	if pageID == "" {
		return
	}
	m.preferences.LastPage = pageID
	m.savePreferences()
}
