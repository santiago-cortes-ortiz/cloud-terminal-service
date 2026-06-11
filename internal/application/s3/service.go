package s3

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	domains3 "aws-terminal/internal/domain/s3"
)

type Service struct {
	objects ObjectStore
}

func NewService(objects ObjectStore) *Service {
	return &Service{objects: objects}
}

type BuildSyncPlanInput struct {
	Profile             string
	Region              string
	Bucket              string
	Prefix              string
	SourcePath          string
	DeleteEnabled       bool
	UploadPlanningMode  domains3.UploadPlanningMode
	StaticWebsitePreset bool
}

func (s *Service) ListBuckets(ctx context.Context, profileName, region string) ([]domains3.Bucket, error) {
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		return nil, fmt.Errorf("profile name is required")
	}

	buckets, err := s.objects.ListBuckets(ctx, profileName, strings.TrimSpace(region))
	if err != nil {
		return nil, err
	}

	sort.Slice(buckets, func(i, j int) bool {
		return buckets[i].Name < buckets[j].Name
	})

	return buckets, nil
}

func (s *Service) InspectSource(sourcePath string) (domains3.SourceSelection, error) {
	sourcePath = strings.TrimSpace(sourcePath)
	if sourcePath == "" {
		return domains3.SourceSelection{}, fmt.Errorf("source path is required")
	}

	absolutePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return domains3.SourceSelection{}, fmt.Errorf("resolve source path: %w", err)
	}

	info, err := os.Stat(absolutePath)
	if err != nil {
		return domains3.SourceSelection{}, fmt.Errorf("inspect source path: %w", err)
	}

	selection := domains3.SourceSelection{Path: absolutePath}
	if !info.IsDir() {
		selection.Kind = domains3.SourceKindFile
		selection.TotalSize = info.Size()
		selection.Files = []domains3.SourceFile{{
			LocalPath:      absolutePath,
			DestinationKey: filepath.ToSlash(filepath.Base(absolutePath)),
			Size:           info.Size(),
		}}
		return selection, nil
	}

	selection.Kind = domains3.SourceKindDirectory
	totalSize := int64(0)
	files := make([]domains3.SourceFile, 0)
	err = filepath.WalkDir(absolutePath, func(currentPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(absolutePath, currentPath)
		if err != nil {
			return err
		}

		files = append(files, domains3.SourceFile{
			LocalPath:      currentPath,
			DestinationKey: filepath.ToSlash(relPath),
			Size:           fileInfo.Size(),
		})
		totalSize += fileInfo.Size()
		return nil
	})
	if err != nil {
		return domains3.SourceSelection{}, fmt.Errorf("walk source directory: %w", err)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].DestinationKey < files[j].DestinationKey
	})

	selection.Files = files
	selection.TotalSize = totalSize
	return selection, nil
}

func (s *Service) BuildSyncPlan(ctx context.Context, input BuildSyncPlanInput) (domains3.SyncPlan, error) {
	profileName := strings.TrimSpace(input.Profile)
	if profileName == "" {
		return domains3.SyncPlan{}, fmt.Errorf("profile name is required")
	}

	bucket := strings.TrimSpace(input.Bucket)
	if bucket == "" {
		return domains3.SyncPlan{}, fmt.Errorf("bucket name is required")
	}

	selection, err := s.InspectSource(input.SourcePath)
	if err != nil {
		return domains3.SyncPlan{}, err
	}

	prefix := normalizePrefix(input.Prefix)
	deleteEnabled := input.DeleteEnabled && selection.Kind == domains3.SourceKindDirectory
	uploadPlanningMode := normalizeUploadPlanningMode(input.UploadPlanningMode)
	remoteObjects, err := s.objects.ListObjects(ctx, profileName, strings.TrimSpace(input.Region), bucket, listObjectsPrefix(prefix))
	if err != nil {
		return domains3.SyncPlan{}, err
	}

	plan := domains3.SyncPlan{
		Profile:             profileName,
		Region:              strings.TrimSpace(input.Region),
		Bucket:              bucket,
		Prefix:              prefix,
		Source:              selection,
		DeleteEnabled:       deleteEnabled,
		UploadPlanningMode:  uploadPlanningMode,
		StaticWebsitePreset: input.StaticWebsitePreset,
	}

	remoteSizes := make(map[string]int64, len(remoteObjects))
	for _, object := range remoteObjects {
		remoteSizes[object.Key] = object.Size
	}

	localKeys := make(map[string]struct{}, len(selection.Files))
	for _, file := range selection.Files {
		key := joinObjectKey(prefix, file.DestinationKey)
		localKeys[key] = struct{}{}

		if uploadPlanningMode == domains3.UploadPlanningModeSizeOnly {
			if remoteSize, ok := remoteSizes[key]; ok && remoteSize == file.Size {
				plan.Skips = append(plan.Skips, domains3.SyncSkip{
					LocalPath: file.LocalPath,
					Key:       key,
					Size:      file.Size,
				})
				continue
			}
		}

		plan.Uploads = append(plan.Uploads, domains3.SyncUpload{
			LocalPath: file.LocalPath,
			Key:       key,
			Size:      file.Size,
			Metadata:  uploadMetadataForKey(key, input.StaticWebsitePreset),
		})
	}

	if plan.DeleteEnabled {
		for _, object := range remoteObjects {
			if !objectKeyInPrefix(object.Key, prefix) {
				continue
			}
			if _, ok := localKeys[object.Key]; ok {
				continue
			}
			plan.Deletes = append(plan.Deletes, domains3.SyncDelete{Key: object.Key, Size: object.Size})
		}
	}

	sort.Slice(plan.Uploads, func(i, j int) bool {
		return plan.Uploads[i].Key < plan.Uploads[j].Key
	})
	sort.Slice(plan.Skips, func(i, j int) bool {
		return plan.Skips[i].Key < plan.Skips[j].Key
	})
	sort.Slice(plan.Deletes, func(i, j int) bool {
		return plan.Deletes[i].Key < plan.Deletes[j].Key
	})

	return plan, nil
}

func (s *Service) ExecuteSync(ctx context.Context, plan domains3.SyncPlan, progress chan<- domains3.SyncProgress) (domains3.SyncResult, error) {
	const (
		maxUploadWorkers   = 8
		maxDeleteBatchSize = 1000
	)

	startedAt := time.Now()
	result := domains3.SyncResult{Skipped: plan.SkipCount()}
	totalSteps := plan.UploadCount()
	if plan.DeleteEnabled {
		totalSteps += plan.DeleteCount()
	}

	totalUploadBytes := int64(0)
	for _, upload := range plan.Uploads {
		totalUploadBytes += upload.Size
	}

	var mu sync.Mutex
	completed := 0
	uploadedBytes := int64(0)
	lastByteEmit := time.Time{}
	emit := func(stage, detail string) {
		if progress == nil {
			return
		}

		mu.Lock()
		snapshot := domains3.SyncProgress{
			Stage:            stage,
			Current:          completed,
			Total:            totalSteps,
			Detail:           detail,
			Uploaded:         result.Uploaded,
			Deleted:          result.Deleted,
			Skipped:          result.Skipped,
			UploadedBytes:    uploadedBytes,
			TotalUploadBytes: totalUploadBytes,
		}
		mu.Unlock()
		progress <- snapshot
	}
	emitByteProgress := func(detail string, bytes int64) {
		if bytes <= 0 {
			return
		}

		mu.Lock()
		uploadedBytes += bytes
		shouldEmit := progress != nil && (time.Since(lastByteEmit) >= 100*time.Millisecond || uploadedBytes >= totalUploadBytes)
		if shouldEmit {
			lastByteEmit = time.Now()
			snapshot := domains3.SyncProgress{
				Stage:            "uploading",
				Current:          completed,
				Total:            totalSteps,
				Detail:           detail,
				Uploaded:         result.Uploaded,
				Deleted:          result.Deleted,
				Skipped:          result.Skipped,
				UploadedBytes:    uploadedBytes,
				TotalUploadBytes: totalUploadBytes,
			}
			mu.Unlock()
			progress <- snapshot
			return
		}
		mu.Unlock()
	}

	if totalSteps == 0 {
		emit("complete", "Nothing to upload or delete.")
		return result, nil
	}

	if len(plan.Uploads) > 0 {
		emit("uploading", fmt.Sprintf("%d files queued", len(plan.Uploads)))

		workerCount := len(plan.Uploads)
		if workerCount > maxUploadWorkers {
			workerCount = maxUploadWorkers
		}

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		jobs := make(chan domains3.SyncUpload)
		errCh := make(chan error, 1)
		var wg sync.WaitGroup

		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for upload := range jobs {
					if err := s.objects.UploadFile(ctx, UploadFileInput{
						Profile:   plan.Profile,
						Region:    plan.Region,
						Bucket:    plan.Bucket,
						Key:       upload.Key,
						LocalPath: upload.LocalPath,
						Metadata:  upload.Metadata,
						Progress: func(bytes int64) {
							emitByteProgress(upload.Key, bytes)
						},
					}); err != nil {
						select {
						case errCh <- fmt.Errorf("upload %s: %w", upload.Key, err):
							cancel()
						default:
						}
						return
					}

					mu.Lock()
					result.Uploaded++
					completed++
					mu.Unlock()
					emit("uploading", upload.Key)
				}
			}()
		}

	uploadLoop:
		for _, upload := range plan.Uploads {
			select {
			case <-ctx.Done():
				break uploadLoop
			case jobs <- upload:
			}
		}
		close(jobs)
		wg.Wait()

		select {
		case err := <-errCh:
			return result, err
		default:
		}
	}

	if plan.DeleteEnabled {
		if len(plan.Deletes) > 0 {
			emit("deleting", fmt.Sprintf("%d remote objects queued", len(plan.Deletes)))
		}
		for start := 0; start < len(plan.Deletes); start += maxDeleteBatchSize {
			end := start + maxDeleteBatchSize
			if end > len(plan.Deletes) {
				end = len(plan.Deletes)
			}

			batchDeletes := plan.Deletes[start:end]
			keys := make([]string, len(batchDeletes))
			for i, deletion := range batchDeletes {
				keys[i] = deletion.Key
			}

			if err := s.objects.DeleteObjects(ctx, DeleteObjectsInput{
				Profile: plan.Profile,
				Region:  plan.Region,
				Bucket:  plan.Bucket,
				Keys:    keys,
			}); err != nil {
				return result, fmt.Errorf("delete %s: %w", deleteBatchDetail(keys), err)
			}

			mu.Lock()
			result.Deleted += len(keys)
			completed += len(keys)
			mu.Unlock()
			emit("deleting", deleteBatchDetail(keys))
		}
	}

	emit("complete", fmt.Sprintf("Sync complete in %s.", time.Since(startedAt).Round(time.Second)))
	return result, nil
}

func uploadMetadataForKey(key string, staticWebsitePreset bool) domains3.UploadMetadata {
	if !staticWebsitePreset {
		return domains3.UploadMetadata{}
	}

	metadata := domains3.UploadMetadata{}
	trimmedKey := strings.TrimSpace(key)
	base := path.Base(trimmedKey)
	if base == "index.html" || strings.HasSuffix(trimmedKey, "/index.html") {
		metadata.CacheControl = "no-cache, no-store, must-revalidate"
	} else if hasHashedAssetName(base) {
		metadata.CacheControl = "public, max-age=31536000, immutable"
	}

	switch {
	case strings.HasSuffix(trimmedKey, ".gz"):
		metadata.ContentEncoding = "gzip"
	case strings.HasSuffix(trimmedKey, ".br"):
		metadata.ContentEncoding = "br"
	}

	return metadata
}

func hasHashedAssetName(base string) bool {
	parts := strings.Split(base, ".")
	if len(parts) < 3 {
		return false
	}
	for _, part := range parts[1 : len(parts)-1] {
		if len(part) >= 8 && isHex(part) {
			return true
		}
	}
	return false
}

func isHex(value string) bool {
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return value != ""
}

func normalizeUploadPlanningMode(mode domains3.UploadPlanningMode) domains3.UploadPlanningMode {
	switch mode {
	case domains3.UploadPlanningModeSizeOnly:
		return domains3.UploadPlanningModeSizeOnly
	default:
		return domains3.UploadPlanningModeFullRefresh
	}
}

func deleteBatchDetail(keys []string) string {
	switch len(keys) {
	case 0:
		return "no objects"
	case 1:
		return keys[0]
	default:
		return fmt.Sprintf("%s + %d more", keys[0], len(keys)-1)
	}
}

func normalizePrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return ""
	}

	parts := strings.FieldsFunc(prefix, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	return strings.Join(parts, "/")
}

func listObjectsPrefix(prefix string) string {
	if prefix == "" {
		return ""
	}
	return prefix + "/"
}

func objectKeyInPrefix(key, prefix string) bool {
	if prefix == "" {
		return true
	}
	return strings.HasPrefix(key, prefix+"/")
}

func joinObjectKey(prefix, suffix string) string {
	suffix = filepath.ToSlash(strings.TrimSpace(suffix))
	suffix = strings.TrimLeft(suffix, "/")
	if prefix == "" {
		return suffix
	}
	if suffix == "" {
		return prefix
	}

	return path.Join(prefix, suffix)
}
