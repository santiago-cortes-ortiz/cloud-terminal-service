package s3

import (
	"fmt"
	"strings"
	"testing"

	domains3 "aws-terminal/internal/domain/s3"
)

func TestReviewStageUsesViewportForLongPlans(t *testing.T) {
	page := NewS3Page(staticS3Service{})
	page.stage = s3StageReview
	page.selectedBucket = "bucket-a"
	page.sourceInfo = &domains3.SourceSelection{Path: "/tmp/source", Kind: domains3.SourceKindDirectory}

	uploads := make([]domains3.SyncUpload, 0, 40)
	for i := 0; i < 40; i++ {
		uploads = append(uploads, domains3.SyncUpload{Key: fmt.Sprintf("assets/file-%02d.js", i)})
	}
	page.plan = &domains3.SyncPlan{Uploads: uploads}

	lines := page.reviewStageLines(80, 20)
	view := strings.Join(lines, "\n")
	if page.reviewViewport.Height != 20 {
		t.Fatalf("expected review viewport height 20, got %d", page.reviewViewport.Height)
	}
	if !strings.Contains(view, "Scroll:") {
		t.Fatalf("expected rendered review to include scroll indicator for long plan, got:\n%s", view)
	}
	if !strings.Contains(view, "assets/file-00.js") {
		t.Fatalf("expected first upload key in viewport, got:\n%s", view)
	}
}
