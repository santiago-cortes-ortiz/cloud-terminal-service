package s3

import (
	"testing"

	domains3 "aws-terminal/internal/domain/s3"
)

func TestSyncPercentUsesUploadBytesWhenAvailable(t *testing.T) {
	got := syncPercent(domains3.SyncProgress{
		Stage:            "uploading",
		Current:          0,
		Total:            10,
		UploadedBytes:    25,
		TotalUploadBytes: 100,
	})
	if got != 0.25 {
		t.Fatalf("syncPercent() = %v, want 0.25", got)
	}
}

func TestSyncPercentFallsBackToStepProgress(t *testing.T) {
	got := syncPercent(domains3.SyncProgress{Current: 2, Total: 4})
	if got != 0.5 {
		t.Fatalf("syncPercent() = %v, want 0.5", got)
	}
}
