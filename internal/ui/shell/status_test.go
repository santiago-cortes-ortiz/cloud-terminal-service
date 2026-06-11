package shell

import (
	"strings"
	"testing"

	"aws-terminal/internal/ui/pages"
)

type statusTestPage struct {
	testPage
	status pages.Status
}

func (p *statusTestPage) PageStatus(pages.State) pages.Status {
	return p.status
}

func TestCurrentPageStatusUsesOptionalProvider(t *testing.T) {
	page := &statusTestPage{testPage: testPage{id: "status"}, status: pages.Status{Message: "workflow running"}}
	model := Model{pageRegistry: []pages.Page{page}, pageList: newSidebarListModel()}
	model.refreshPageList("status")

	status := model.currentPageStatus()
	if status.Message != "workflow running" {
		t.Fatalf("expected workflow status, got %#v", status)
	}
}

func TestFooterIncludesPageStatusSeparatelyFromGlobalStatus(t *testing.T) {
	page := &statusTestPage{testPage: testPage{id: "status"}, status: pages.Status{Message: "workflow running"}}
	model := Model{
		width:         120,
		height:        40,
		pageRegistry:  []pages.Page{page},
		pageList:      newSidebarListModel(),
		profileList:   newSidebarListModel(),
		regionList:    newSidebarListModel(),
		statusMessage: "global auth status",
	}
	model.refreshPageList("status")

	footer := model.footerView(120)
	if !strings.Contains(footer, "Page status: workflow running") {
		t.Fatalf("expected page status in footer, got:\n%s", footer)
	}
	if strings.Contains(footer, "global auth status") {
		t.Fatalf("did not expect global shell status to be mixed into footer status parts, got:\n%s", footer)
	}
}
