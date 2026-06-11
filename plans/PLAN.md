# Plan: Diagnose and fix S3 sync upload/delete behavior

## Context

The S3 sync workflow should behave like `aws s3 sync <local-dir> s3://<bucket>/ --delete --profile <profile>`: upload local files under the expected object keys and optionally delete remote objects that no longer exist locally. The current issue appears in `internal/infrastructure/awss3/`, but the sync behavior spans application planning/execution and the AWS S3 adapter.

Findings:
- The reported browser failure is MIME-type related after upload + CloudFront invalidation, not primarily auth/profile/region selection.
- `internal/infrastructure/awss3/store.go` uses AWS SDK v2 `PutObject`, `DeleteObject`, `ListObjectsV2`, and bucket-region resolution via `HeadBucket`.
- `UploadFile` only sets `ContentType` when Go's `mime.TypeByExtension` knows the extension. A local check shows Go returns empty content types for some common frontend artifacts such as `.webmanifest` and `.map`.
- `internal/application/s3/service.go` currently skips uploads when remote size equals local size. This can leave stale object content and, importantly for this bug, stale/wrong S3 metadata such as `Content-Type` even after CloudFront invalidation.
- Existing profile/region/delete toggles are already wired in the UI/application flow.

## Approach

Make S3 uploads safer for static frontend deployments while keeping the implementation simple:
- Keep the existing directory-to-key behavior, which already mirrors `aws s3 sync dist/.../browser s3://bucket/` by walking the selected directory and using paths relative to that directory.
- Add an explicit frontend-focused content-type fallback in the S3 adapter for extensions Go may not know, while still using `mime.TypeByExtension` as the first choice.
- Stop using remote size alone as proof that an upload can be skipped. Prefer correctness over a potentially unsafe optimization for this TUI workflow. The simplest fix is to plan uploads for all local files so existing objects get fresh content and metadata; delete remains controlled by the existing toggle.
- Keep delete planning scoped to objects returned by `ListObjectsV2` for the selected prefix.

## Files to modify

Files:
- `internal/application/s3/service.go`
- `internal/infrastructure/awss3/store.go`

Test files to add/update:
- `internal/application/s3/service_test.go`
- `internal/infrastructure/awss3/store_test.go` for content-type helper coverage if the helper is kept testable without AWS calls.

## Reuse

Existing reusable code/patterns found:
- `InspectSource`, `normalizePrefix`, and `joinObjectKey` in `internal/application/s3/service.go` for source walking and key construction.
- `ObjectStore` interface in `internal/application/s3/ports.go` for unit testing sync planning/execution without hitting AWS.
- Existing S3 adapter methods in `internal/infrastructure/awss3/store.go`: `ListObjects`, `UploadFile`, `DeleteObject`, and bucket-region-aware `bucketClient`.

## Steps

- [x] Extract/test content-type resolution in `internal/infrastructure/awss3/store.go`: use `mime.TypeByExtension` first, then explicit fallbacks for common static-web extensions such as `.webmanifest`, `.map`, `.ico`, `.xml`, `.txt`, and any other missing frontend types found during implementation.
- [x] Update `UploadFile` to use the content-type helper so every relevant frontend artifact gets a correct `Content-Type` on S3.
- [x] Update sync planning in `internal/application/s3/service.go` so local files are uploaded even when the remote object has the same size, avoiding stale content/metadata from the current size-only skip logic.
- [x] Preserve delete toggle behavior: only delete when enabled and only for remote objects under the selected prefix that are absent locally.
- [x] Add unit tests for AWS CLI-like key behavior for syncing a directory to bucket root and to an optional prefix.
- [x] Add unit tests proving same-size remote objects are planned for upload rather than skipped.
- [x] Add unit tests for delete planning under the selected prefix.

## Verification

- Run `go test ./...`.
- Manually sync a small temp directory to a test bucket/prefix with delete disabled and enabled.
- Compare resulting S3 keys with `aws s3 sync <dir> s3://<bucket>/<prefix> --delete --profile <profile>` for the same fixture.
