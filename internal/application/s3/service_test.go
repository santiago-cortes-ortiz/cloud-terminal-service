package s3

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	domains3 "aws-terminal/internal/domain/s3"
)

type fakeObjectStore struct {
	listObjectsPrefix string
	remoteObjects     []domains3.RemoteObject
	uploadProgress    map[string][]int64
	deleteBatches     [][]string
}

func (f *fakeObjectStore) ListBuckets(context.Context, string, string) ([]domains3.Bucket, error) {
	return nil, nil
}

func (f *fakeObjectStore) ListObjects(_ context.Context, _, _, _, prefix string) ([]domains3.RemoteObject, error) {
	f.listObjectsPrefix = prefix
	return f.remoteObjects, nil
}

func (f *fakeObjectStore) UploadFile(_ context.Context, input UploadFileInput) error {
	for _, bytes := range f.uploadProgress[input.Key] {
		if input.Progress != nil {
			input.Progress(bytes)
		}
	}
	return nil
}
func (f *fakeObjectStore) DeleteObject(context.Context, DeleteObjectInput) error { return nil }
func (f *fakeObjectStore) DeleteObjects(_ context.Context, input DeleteObjectsInput) error {
	f.deleteBatches = append(f.deleteBatches, append([]string(nil), input.Keys...))
	return nil
}

func TestBuildSyncPlanDirectoryKeysMatchAWSSyncToBucketRoot(t *testing.T) {
	source := testSourceDir(t, map[string]string{
		"index.html":       "<html></html>",
		"assets/app.js":    "console.log('ok')",
		"assets/style.css": "body{}",
	})
	store := &fakeObjectStore{}
	service := NewService(store)

	plan, err := service.BuildSyncPlan(context.Background(), BuildSyncPlanInput{
		Profile:    "pre",
		Region:     "us-east-1",
		Bucket:     "example-bucket",
		SourcePath: source,
	})
	if err != nil {
		t.Fatalf("BuildSyncPlan() error = %v", err)
	}

	got := uploadKeys(plan.Uploads)
	want := []string{"assets/app.js", "assets/style.css", "index.html"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("upload keys = %#v, want %#v", got, want)
	}
	if store.listObjectsPrefix != "" {
		t.Fatalf("ListObjects prefix = %q, want bucket root", store.listObjectsPrefix)
	}
}

func TestBuildSyncPlanDirectoryKeysMatchAWSSyncToPrefix(t *testing.T) {
	source := testSourceDir(t, map[string]string{
		"index.html":    "<html></html>",
		"assets/app.js": "console.log('ok')",
	})
	store := &fakeObjectStore{}
	service := NewService(store)

	plan, err := service.BuildSyncPlan(context.Background(), BuildSyncPlanInput{
		Profile:    "pre",
		Region:     "us-east-1",
		Bucket:     "example-bucket",
		Prefix:     "/preview/site/",
		SourcePath: source,
	})
	if err != nil {
		t.Fatalf("BuildSyncPlan() error = %v", err)
	}

	got := uploadKeys(plan.Uploads)
	want := []string{"preview/site/assets/app.js", "preview/site/index.html"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("upload keys = %#v, want %#v", got, want)
	}
	if store.listObjectsPrefix != "preview/site/" {
		t.Fatalf("ListObjects prefix = %q, want %q", store.listObjectsPrefix, "preview/site/")
	}
}

func TestBuildSyncPlanDefaultsToFullRefresh(t *testing.T) {
	source := testSourceDir(t, map[string]string{"main.js": "12345"})
	store := &fakeObjectStore{remoteObjects: []domains3.RemoteObject{{Key: "main.js", Size: 5}}}
	service := NewService(store)

	plan, err := service.BuildSyncPlan(context.Background(), BuildSyncPlanInput{
		Profile:    "pre",
		Bucket:     "example-bucket",
		SourcePath: source,
	})
	if err != nil {
		t.Fatalf("BuildSyncPlan() error = %v", err)
	}
	if got := uploadKeys(plan.Uploads); !reflect.DeepEqual(got, []string{"main.js"}) {
		t.Fatalf("upload keys = %#v, want same-size file to be uploaded", got)
	}
	if len(plan.Skips) != 0 {
		t.Fatalf("skips = %#v, want none", plan.Skips)
	}
	if plan.UploadPlanningMode != domains3.UploadPlanningModeFullRefresh {
		t.Fatalf("planning mode = %q", plan.UploadPlanningMode)
	}
}

func TestBuildSyncPlanStaticWebsitePresetAppliesUploadMetadata(t *testing.T) {
	source := testSourceDir(t, map[string]string{
		"index.html":             "html",
		"assets/app.abcdef12.js": "js",
		"assets/app.js.gz":       "gzip",
	})
	store := &fakeObjectStore{}
	service := NewService(store)

	plan, err := service.BuildSyncPlan(context.Background(), BuildSyncPlanInput{
		Profile:             "pre",
		Bucket:              "example-bucket",
		SourcePath:          source,
		StaticWebsitePreset: true,
	})
	if err != nil {
		t.Fatalf("BuildSyncPlan() error = %v", err)
	}

	metadataByKey := map[string]domains3.UploadMetadata{}
	for _, upload := range plan.Uploads {
		metadataByKey[upload.Key] = upload.Metadata
	}
	if got := metadataByKey["index.html"].CacheControl; got != "no-cache, no-store, must-revalidate" {
		t.Fatalf("index cache-control = %q", got)
	}
	if got := metadataByKey["assets/app.abcdef12.js"].CacheControl; got != "public, max-age=31536000, immutable" {
		t.Fatalf("asset cache-control = %q", got)
	}
	if got := metadataByKey["assets/app.js.gz"].ContentEncoding; got != "gzip" {
		t.Fatalf("gzip content-encoding = %q", got)
	}
	if !plan.StaticWebsitePreset {
		t.Fatal("expected plan to record static website preset")
	}
}

func TestBuildSyncPlanOptimizedModeSkipsSameSizeRemoteObject(t *testing.T) {
	source := testSourceDir(t, map[string]string{"main.js": "12345"})
	store := &fakeObjectStore{remoteObjects: []domains3.RemoteObject{{Key: "main.js", Size: 5}}}
	service := NewService(store)

	plan, err := service.BuildSyncPlan(context.Background(), BuildSyncPlanInput{
		Profile:            "pre",
		Bucket:             "example-bucket",
		SourcePath:         source,
		UploadPlanningMode: domains3.UploadPlanningModeSizeOnly,
	})
	if err != nil {
		t.Fatalf("BuildSyncPlan() error = %v", err)
	}
	if len(plan.Uploads) != 0 {
		t.Fatalf("uploads = %#v, want none", plan.Uploads)
	}
	if got := skipKeys(plan.Skips); !reflect.DeepEqual(got, []string{"main.js"}) {
		t.Fatalf("skip keys = %#v, want main.js", got)
	}
	if plan.UploadPlanningMode != domains3.UploadPlanningModeSizeOnly {
		t.Fatalf("planning mode = %q", plan.UploadPlanningMode)
	}
}

func TestBuildSyncPlanOptimizedModeUploadsDifferentSizeRemoteObject(t *testing.T) {
	source := testSourceDir(t, map[string]string{"main.js": "12345"})
	store := &fakeObjectStore{remoteObjects: []domains3.RemoteObject{{Key: "main.js", Size: 4}}}
	service := NewService(store)

	plan, err := service.BuildSyncPlan(context.Background(), BuildSyncPlanInput{
		Profile:            "pre",
		Bucket:             "example-bucket",
		SourcePath:         source,
		UploadPlanningMode: domains3.UploadPlanningModeSizeOnly,
	})
	if err != nil {
		t.Fatalf("BuildSyncPlan() error = %v", err)
	}
	if got := uploadKeys(plan.Uploads); !reflect.DeepEqual(got, []string{"main.js"}) {
		t.Fatalf("upload keys = %#v, want main.js", got)
	}
	if len(plan.Skips) != 0 {
		t.Fatalf("skips = %#v, want none", plan.Skips)
	}
}

func TestBuildSyncPlanDeletePlanningUnderSelectedPrefix(t *testing.T) {
	source := testSourceDir(t, map[string]string{"index.html": "local"})
	store := &fakeObjectStore{remoteObjects: []domains3.RemoteObject{
		{Key: "site/index.html", Size: 5},
		{Key: "site/stale.js", Size: 10},
		{Key: "site/nested/old.css", Size: 11},
		{Key: "site-other/keep.js", Size: 12},
	}}
	service := NewService(store)

	plan, err := service.BuildSyncPlan(context.Background(), BuildSyncPlanInput{
		Profile:       "pre",
		Bucket:        "example-bucket",
		Prefix:        "site",
		SourcePath:    source,
		DeleteEnabled: true,
	})
	if err != nil {
		t.Fatalf("BuildSyncPlan() error = %v", err)
	}

	got := deleteKeys(plan.Deletes)
	want := []string{"site/nested/old.css", "site/stale.js"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("delete keys = %#v, want %#v", got, want)
	}
}

func TestExecuteSyncEmitsByteProgress(t *testing.T) {
	store := &fakeObjectStore{uploadProgress: map[string][]int64{
		"app.js": []int64{2, 3},
	}}
	service := NewService(store)
	progressCh := make(chan domains3.SyncProgress, 16)

	result, err := service.ExecuteSync(context.Background(), domains3.SyncPlan{
		Profile: "pre",
		Bucket:  "example-bucket",
		Uploads: []domains3.SyncUpload{{Key: "app.js", LocalPath: "app.js", Size: 5}},
	}, progressCh)
	close(progressCh)
	if err != nil {
		t.Fatalf("ExecuteSync() error = %v", err)
	}
	if result.Uploaded != 1 {
		t.Fatalf("uploaded = %d, want 1", result.Uploaded)
	}

	var sawBytes bool
	for progress := range progressCh {
		if progress.UploadedBytes == 5 && progress.TotalUploadBytes == 5 {
			sawBytes = true
		}
	}
	if !sawBytes {
		t.Fatal("expected byte progress with 5/5 bytes")
	}
}

func TestExecuteSyncDeletesInBatches(t *testing.T) {
	store := &fakeObjectStore{}
	service := NewService(store)
	deletes := make([]domains3.SyncDelete, 1001)
	for i := range deletes {
		deletes[i] = domains3.SyncDelete{Key: fmt.Sprintf("old-%04d.js", i)}
	}

	result, err := service.ExecuteSync(context.Background(), domains3.SyncPlan{
		Profile:       "pre",
		Bucket:        "example-bucket",
		DeleteEnabled: true,
		Deletes:       deletes,
	}, nil)
	if err != nil {
		t.Fatalf("ExecuteSync() error = %v", err)
	}
	if result.Deleted != 1001 {
		t.Fatalf("deleted = %d, want 1001", result.Deleted)
	}
	if len(store.deleteBatches) != 2 {
		t.Fatalf("delete batches = %d, want 2", len(store.deleteBatches))
	}
	if len(store.deleteBatches[0]) != 1000 || len(store.deleteBatches[1]) != 1 {
		t.Fatalf("delete batch sizes = %d/%d, want 1000/1", len(store.deleteBatches[0]), len(store.deleteBatches[1]))
	}
}

func TestBuildSyncPlanDoesNotDeleteWhenToggleDisabled(t *testing.T) {
	source := testSourceDir(t, map[string]string{"index.html": "local"})
	store := &fakeObjectStore{remoteObjects: []domains3.RemoteObject{{Key: "stale.js", Size: 10}}}
	service := NewService(store)

	plan, err := service.BuildSyncPlan(context.Background(), BuildSyncPlanInput{
		Profile:       "pre",
		Bucket:        "example-bucket",
		SourcePath:    source,
		DeleteEnabled: false,
	})
	if err != nil {
		t.Fatalf("BuildSyncPlan() error = %v", err)
	}
	if len(plan.Deletes) != 0 {
		t.Fatalf("deletes = %#v, want none", plan.Deletes)
	}
}

func testSourceDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		filePath := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}
	return dir
}

func uploadKeys(uploads []domains3.SyncUpload) []string {
	keys := make([]string, len(uploads))
	for i, upload := range uploads {
		keys[i] = upload.Key
	}
	return keys
}

func deleteKeys(deletes []domains3.SyncDelete) []string {
	keys := make([]string, len(deletes))
	for i, deletion := range deletes {
		keys[i] = deletion.Key
	}
	return keys
}

func skipKeys(skips []domains3.SyncSkip) []string {
	keys := make([]string, len(skips))
	for i, skip := range skips {
		keys[i] = skip.Key
	}
	return keys
}
