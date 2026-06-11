package shell

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"

	domainregion "aws-terminal/internal/domain/region"
)

type regionListItem struct {
	Region domainregion.Region
	Active bool
}

func (i regionListItem) FilterValue() string {
	return strings.TrimSpace(i.Region.ID + " " + i.Region.Name)
}

func (i regionListItem) SidebarLabel() string {
	marker := " "
	if i.Active {
		marker = "●"
	}

	return fmt.Sprintf("%s %s", marker, i.Region.ID)
}

func (i regionListItem) SidebarActive() bool {
	return i.Active
}

func (m *Model) refreshRegionList(targetRegion string) {
	if strings.TrimSpace(targetRegion) == "" {
		if item, ok := m.regionList.SelectedItem().(regionListItem); ok {
			targetRegion = item.Region.ID
		}
	}

	items := make([]list.Item, 0, len(m.regions))
	for _, region := range m.regions {
		items = append(items, regionListItem{
			Region: region,
			Active: region.ID == m.activeRegion(),
		})
	}
	_ = m.regionList.SetItems(items)

	if strings.TrimSpace(targetRegion) != "" {
		m.selectRegionListItem(targetRegion)
	} else if len(items) > 0 {
		m.regionList.ResetSelected()
	}

	m.applySidebarListFocus()
	m.syncSidebarListsLayout()
}

func (m *Model) selectRegionListItem(regionID string) {
	if index, found := m.regionIndex(strings.TrimSpace(regionID)); found {
		m.regionList.Select(index)
		return
	}

	if len(m.regions) > 0 {
		m.regionList.ResetSelected()
	}
}

func (m *Model) syncRegionSelection() {
	selected := ""
	for _, preferredRegion := range m.preferredRegionNames() {
		preferredRegion = strings.TrimSpace(preferredRegion)
		if preferredRegion == "" {
			continue
		}
		m.ensureRegionAvailable(preferredRegion)
		selected = preferredRegion
		break
	}

	if selected == "" && len(m.regions) > 0 {
		selected = m.regions[0].ID
	}

	m.selectedRegion = selected
	m.refreshRegionList(selected)
}
