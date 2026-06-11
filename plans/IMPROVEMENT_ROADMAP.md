# Incremental Improvement Roadmap

## Context

The TUI is already structured with a clean `domain -> application -> infrastructure/ui` separation and has working AWS profile/SSO, S3 sync, and CloudFront invalidation workflows. The next goal is to improve robustness, scalability, performance, and maintainability gradually.

This roadmap is intentionally sequential: implement one improvement, test it, manually verify the TUI still behaves correctly, then continue. If an improvement reveals regressions or design friction, pause and fix/refactor before moving to the next item.

## Approach

Use small, reversible changes. Prefer tests at the application and UI-update/state-transition layers before larger refactors. Keep live AWS calls out of unit tests. Maintain the existing user workflows while improving internals.

Recommended order prioritizes correctness and architecture risks first, then performance, then UX and polish.

## Files likely to change over time

Core shell/page architecture:
- `internal/ui/shell/model.go`
- `internal/ui/shell/update.go`
- `internal/ui/shell/messages.go`
- `internal/ui/pages/page.go`
- `internal/ui/pages/s3.go`
- `internal/ui/pages/cloudfront.go`
- `internal/ui/pages/registry.go`

AWS/application services:
- `internal/application/s3/service.go`
- `internal/application/s3/ports.go`
- `internal/infrastructure/awss3/store.go`
- `internal/application/cloudfront/service.go`
- `internal/infrastructure/awscloudfront/service.go`
- `internal/infrastructure/awssso/device_flow.go`
- `internal/infrastructure/awsconfig/*`

Future/new support packages may include:
- `internal/ui/pages/s3/*` or split `s3_*.go` files
- `internal/ui/pages/cloudfront/*` or split `cloudfront_*.go` files
- `internal/ui/workflow`
- `internal/infrastructure/awsclients`
- `internal/config` or similar for persisted app preferences

## Reuse

Existing patterns/utilities to keep using:
- Page interface in `internal/ui/pages/page.go`.
- Shell focus routing in `internal/ui/shell/update.go`.
- Session state projection via `pages.State` in `internal/ui/shell/model.go`.
- Application ports for AWS boundaries in `internal/application/*/ports.go`.
- S3 sync planning/execution tests in `internal/application/s3/service_test.go`.
- Content type helper tests in `internal/infrastructure/awss3/store_test.go`.
- Shared style/render helpers in `internal/ui/styles/theme.go`.
- Sidebar/list delegate pattern in `internal/ui/shell/sidebar_list.go`.

## Roadmap steps

### 1. Route async page messages to their owning page

Problem: shell currently forwards many non-shell messages only to `currentPage()`. If a user starts S3 sync and switches to CloudFront, S3 progress messages can be ignored by the wrong page.

Plan:
- Add page ownership metadata to page-specific async messages, or introduce a small `PageMsg` wrapper with `PageID`.
- Update shell message routing so page-owned messages are delivered to the matching registered page.
- Keep visible-page rendering unchanged.
- Add tests for switching pages while an S3/CloudFront async message is in flight.

Verification:
- `go test ./...`
- Manual: start S3 bucket load/sync, switch page, confirm operation state still updates when returning.

### 2. Add cancellable contexts for long-running commands

Problem: commands use `context.Background()` and continue after session/page changes or quit.

Plan:
- Introduce command-scoped contexts/cancel funcs in shell/page state.
- Cancel stale profile/session/page operations when session key changes or page resets.
- Add explicit cancellation handling for S3 sync and CloudFront polling.
- Ensure cancellation errors render as user-friendly statuses.

Verification:
- Unit tests for cancellation state transitions where practical.
- Manual: start long operation, switch profile/region or cancel, confirm no stale completion overwrites current state.

### 3. Add explicit cancel/back behavior

Problem: long workflows have limited cancellation semantics; `q` quits globally and `b` backs up in some stages.

Plan:
- Define page-level cancel keys, likely `esc` for cancel/back and `b` for previous step where already used.
- S3: allow cancelling sync if running after step 2 introduces cancellable contexts.
- CloudFront: allow cancelling invalidation polling view without cancelling already-created AWS invalidation.
- Make help text reflect cancel behavior.

Verification:
- `go test ./...`
- Manual: cancel S3 source/prefix/review/sync stages and CloudFront polling.

### 4. Standardize page key maps

Problem: several pages use raw `msg.String()` checks while help bindings live separately.

Plan:
- Create key map structs for S3 and CloudFront pages.
- Replace raw string matching with `key.Matches`.
- Ensure `ShortHelp`/`FullHelp` are derived from the same bindings.

Verification:
- `go test ./...`
- Manual keyboard pass through each page stage.

### 5. Split large page files

Problem: `s3.go` and `cloudfront.go` combine model, update, view, commands, and keys.

Plan:
- Split S3 into focused files such as `s3_model.go`, `s3_update.go`, `s3_view.go`, `s3_commands.go`, `s3_keys.go` inside `internal/ui/pages` initially.
- Split CloudFront similarly.
- Avoid behavioral changes in this step.

Verification:
- `go test ./...`
- `go build ./...`
- Manual smoke test: focus, S3, CloudFront flows still render and navigate.

### 6. Add UI state-transition tests

Problem: most tests are application-layer; shell/page transitions are mostly untested.

Plan:
- Add tests for shell focus cycling and page selection.
- Add tests for S3 page session reset and stage transitions.
- Add tests for CloudFront polling lifecycle state transitions.
- Use fake services; no live AWS calls.

Verification:
- `go test ./internal/ui/... ./...`

### 7. Add viewport support for long page content

Problem: long review plans and future AWS resource forms can overflow fixed-height boxes.

Plan:
- Add `bubbles/viewport` for long detail/review content.
- Start with S3 review screen upload/delete samples or full plan list.
- Consider a reusable viewport wrapper for future resource pages.
- Keep compact layouts working for narrow terminals.

Verification:
- Manual: small and large terminal sizes; large S3 plans scroll cleanly.
- `go test ./...`

### 8. Separate page workflow status from global shell status

Problem: global shell status and page-local status are mixed conceptually.

Plan:
- Define page-local status/error conventions.
- Footer may display global status plus current page status.
- Avoid pages relying on shell status strings except for global auth/session state.

Verification:
- Manual: auth errors, S3 errors, CloudFront errors display in the expected area.
- `go test ./...`

### 9. Introduce reusable workflow primitives

Problem: IAM/EC2/Networking will likely repeat staged workflow patterns.

Plan:
- Extract small helpers only after S3/CloudFront patterns are stable.
- Candidates: async load state, staged workflow labels, review/action summaries, session-key reset helper.
- Keep generic helpers small; avoid over-abstracting page-specific behavior.

Verification:
- Existing S3/CloudFront behavior unchanged.
- New helper tests where logic is non-trivial.

### 10. Add shared AWS config/client factory

Problem: S3 caches clients, CloudFront creates new clients, retry/region behavior is scattered.

Plan:
- Create `internal/infrastructure/awsclients` or similar.
- Centralize AWS config loading, client caching, default region fallback, user-agent, retry/timeout options.
- Migrate S3 and CloudFront adapters to use it.

Verification:
- Unit tests for cache keys/fallback behavior where practical.
- Manual AWS smoke test for S3 list and CloudFront list.
- `go test ./...`

### 11. Improve S3 bucket-region cache keying

Problem: S3 bucket region cache is keyed only by bucket name.

Plan:
- Key bucket region cache by profile/account/partition/bucket where available, or at minimum `profile|bucket`.
- Keep behavior compatible with existing bucket-region resolution.

Verification:
- Unit test cache key helper if extracted.
- Manual: list/sync buckets across two profiles if available.

### 12. Add configurable AWS timeouts/retries

Problem: AWS SDK default retry behavior is invisible to the TUI.

Plan:
- Add app-level constants or config for operation timeouts.
- Set SDK retry/user-agent options centrally in the shared AWS client factory.
- Surface retrying/throttling messages where feasible.

Verification:
- Unit tests for config construction if practical.
- Manual: network interruption/throttling behavior does not freeze UI indefinitely.

### 13. Cache AWS SSO client registrations

Problem: native SSO registers a new OIDC client each login.

Plan:
- Store registration cache compatible with AWS SSO expectations or app-specific cache.
- Key by SSO start URL/session/region/scopes.
- Reuse while valid; refresh when expired.

Verification:
- Unit tests around cache read/write/expiry.
- Manual: repeated SSO login starts faster and still writes token cache correctly.

### 14. Use S3 multipart uploader for large files

Problem: `PutObject` is simple but less robust for large files.

Plan:
- Replace upload implementation with AWS SDK `feature/s3/manager.Uploader`.
- Preserve content-type and metadata setting.
- Tune concurrency/part size conservatively.

Verification:
- Unit tests for request option mapping where possible.
- Manual: upload a large file and confirm progress/error handling.
- `go test ./...`

### 15. Add byte-level S3 progress

Problem: current sync progress is per object; large files can look stuck.

Plan:
- Wrap file readers to emit byte progress.
- Extend domain progress model with byte totals/current file progress.
- Render upload speed/ETA in S3 sync stage.

Verification:
- Unit tests for progress reader.
- Manual: upload large file and confirm progress moves during file upload.

### 16. Batch S3 deletes

Problem: deletes use one `DeleteObject` call per key.

Plan:
- Add `DeleteObjects` batch port or update existing delete execution to batch internally.
- Use batches of up to 1000 keys.
- Report partial failures clearly.

Verification:
- Application tests for batching and partial failure behavior.
- Manual: delete-enabled sync with many stale objects.

### 17. Add optional smarter S3 upload planning

Problem: uploading all local files is safe but can be expensive.

Plan:
- Keep current safe behavior as default initially.
- Add explicit mode/setting for optimized planning using size and optionally ETag/checksum/metadata.
- Make UI explain the tradeoff clearly.

Verification:
- Tests for full-refresh mode and optimized mode.
- Manual: repeated sync of large directory in both modes.

### 18. Add static-site S3 metadata presets

Problem: frontend deployments often need cache-control and content-encoding behavior.

Plan:
- Add upload metadata options to application/domain model.
- Implement static website preset:
  - `index.html`: no-cache
  - hashed assets: long cache immutable
  - `.gz`/`.br`: content-encoding
- Show metadata preset on review screen.

Verification:
- Unit tests for metadata decision helper.
- Manual: inspect uploaded object metadata in AWS/S3 CLI.

### 19. Add safer destructive-action confirmation

Problem: delete toggle is explicit, but many deletions may deserve stronger confirmation.

Plan:
- If delete count exceeds a threshold, require typed confirmation such as `DELETE`.
- Reuse pattern later for IAM/EC2 destructive actions.

Verification:
- UI tests for confirmation stage transitions.
- Manual: small delete and large delete flows.

### 20. Add persisted preferences

Problem: app forgets last profile/region/page/source directory.

Plan:
- Add small config store, likely under `~/.config/aws-terminal/config.json`.
- Persist last profile, region, page, S3 source dir, recent prefixes/paths.
- Keep failures non-fatal.

Verification:
- Unit tests using temp config directory/env override.
- Manual: restart app and confirm selections restore.

### 21. Add command palette / quick actions

Problem: sidebar navigation may become slower as pages/actions grow.

Plan:
- Add command palette key such as `ctrl+p` or `/`.
- Start with page navigation and common actions.
- Later include resource-specific actions.

Verification:
- Manual: open palette, fuzzy/select action, confirm focus/page changes.
- `go test ./...`

### 22. Add fake AWS integration contracts

Problem: infrastructure request mapping can regress without live AWS tests.

Plan:
- Where AWS SDK clients can be wrapped behind small interfaces, add request mapping tests.
- Continue avoiding live AWS in unit tests.
- Cover CloudFront invalidation input, S3 upload metadata, S3 delete batching.

Verification:
- `go test ./internal/infrastructure/...`

### 23. Optimize render hot paths only after behavior stabilizes

Problem: current rendering recomputes many strings every frame; fine for now, but may matter with heavier pages.

Plan:
- Profile/render-measure if UI becomes sluggish.
- Keep `View()` pure and avoid I/O/computation there.
- Cache expensive static sections only if measurable.

Verification:
- Manual responsiveness check on large profiles/resource lists.

## Implementation checklist

- [x] 1. Route async page messages to owning page.
- [x] 2. Add cancellable contexts for long-running commands.
- [x] 3. Add explicit cancel/back behavior.
- [x] 4. Standardize page key maps.
- [x] 5. Split large page files without behavior changes.
- [x] 6. Add UI state-transition tests.
- [x] 7. Add viewport support for long content.
- [x] 8. Separate page workflow status from global shell status.
- [x] 9. Introduce reusable workflow primitives.
- [x] 10. Add shared AWS config/client factory.
- [x] 11. Improve S3 bucket-region cache keying.
- [x] 12. Add configurable AWS timeouts/retries.
- [x] 13. Cache AWS SSO client registrations.
- [x] 14. Use S3 multipart uploader for large files.
- [x] 15. Add byte-level S3 progress.
- [x] 16. Batch S3 deletes.
- [x] 17. Add optional smarter S3 upload planning.
- [x] 18. Add static-site S3 metadata presets.
- [x] 19. Add safer destructive-action confirmation.
- [x] 20. Add persisted preferences.
- [x] 21. Add command palette / quick actions.
- [x] 22. Add fake AWS integration contracts.
- [x] 23. Optimize render hot paths only if needed.

## Verification cadence

After each roadmap item:

1. Run targeted tests for touched packages.
2. Run full tests:

   ```bash
   go test ./...
   ```

3. Run build:

   ```bash
   go build ./...
   ```

4. Manual smoke test:
   - Launch app with `go run ./cmd/aws-terminal`.
   - Switch focus among Profiles, Regions, Pages.
   - Authenticate a profile if needed.
   - Open Dashboard, S3, CloudFront.
   - Confirm no stale async messages, broken focus, or rendering regressions.

5. For AWS-touching steps, manually verify with real configured profiles/resources.

## Stop/go rule

Proceed to the next roadmap item only when:

- tests pass,
- build passes,
- the relevant manual workflow works,
- and no new architectural inconsistency was introduced.

If any item fails verification, pause the roadmap and fix/refactor that item before continuing.
