package shell

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"

	domainprofile "aws-terminal/internal/domain/profile"
)

type profileListItem struct {
	Profile domainprofile.Profile
	Active  bool
	Busy    bool
}

func (i profileListItem) FilterValue() string {
	return i.Profile.Name
}

func (i profileListItem) SidebarLabel() string {
	marker := " "
	if i.Busy {
		marker = "…"
	} else if i.Active {
		marker = "●"
	}

	return fmt.Sprintf("%s %s", marker, i.Profile.Name)
}

func (i profileListItem) SidebarActive() bool {
	return i.Active
}

func (m *Model) refreshProfileList(targetName string) {
	if strings.TrimSpace(targetName) == "" {
		if item, ok := m.profileList.SelectedItem().(profileListItem); ok {
			targetName = item.Profile.Name
		}
	}
	busyProfile := m.busyProfileName()

	items := make([]list.Item, 0, len(m.profiles))
	for _, profile := range m.profiles {
		items = append(items, profileListItem{
			Profile: profile,
			Active:  profile.Name == m.activeProfileName(),
			Busy:    profile.Name == busyProfile,
		})
	}
	_ = m.profileList.SetItems(items)

	if strings.TrimSpace(targetName) != "" {
		m.selectProfileListItem(targetName)
	} else if len(items) > 0 {
		m.profileList.ResetSelected()
	}

	m.applySidebarListFocus()
	m.syncSidebarListsLayout()
}

func (m *Model) syncProfileSelection() {
	selected := ""
	for _, preferredName := range m.preferredProfileNames() {
		if preferredName == "" {
			continue
		}
		for _, current := range m.profiles {
			if current.Name == preferredName {
				selected = preferredName
				break
			}
		}
		if selected != "" {
			break
		}
	}
	if selected == "" && len(m.profiles) > 0 {
		selected = m.profiles[0].Name
	}

	m.refreshProfileList(selected)
}

func (m *Model) selectProfileListItem(profileName string) {
	for index, profile := range m.profiles {
		if profile.Name == strings.TrimSpace(profileName) {
			m.profileList.Select(index)
			return
		}
	}

	if len(m.profiles) > 0 {
		m.profileList.ResetSelected()
	}
}

func (m Model) busyProfileName() string {
	if m.authPrompt != nil && strings.TrimSpace(m.authPrompt.ProfileName) != "" {
		return strings.TrimSpace(m.authPrompt.ProfileName)
	}
	if !m.profileBusy {
		return ""
	}
	if profile, ok := m.selectedProfile(); ok {
		return profile.Name
	}

	return ""
}
