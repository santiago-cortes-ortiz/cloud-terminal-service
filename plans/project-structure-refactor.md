# Project Structure Refactor Plan

## Context

The current project structure is already generally healthy: it separates command entrypoint, app wiring, domain types, application use cases, infrastructure adapters, and Bubble Tea UI packages.

The main structural issue found during review is that `internal/ui/shell` is coupled to concrete page implementations (`s3` and `cloudfront`) through its constructors. As more AWS resource pages are added, this will make the shell constructor grow and force shell code to change for every new page.

The intended outcome is to keep the shell generic, move concrete page composition into the app wiring layer, and remove placeholder pages that do not represent implemented workflows yet.

## Approach

Refactor page wiring so `internal/app` builds the page registry and passes it into the shell. The shell should depend only on the shared page abstraction, not on concrete S3 or CloudFront page packages.

Keep the existing layered architecture intact:

```text
cmd/aws-terminal
internal/app
internal/domain
internal/application
internal/infrastructure
internal/ui
```

Avoid a large feature-folder migration for now. The current structure is appropriate; the recommended change is a targeted decoupling of shell from concrete pages.

Target shape:

```text
internal/app/
  app.go                 # dependency wiring and Bubble Tea program startup
  pages.go               # app-specific page registry construction

internal/ui/
  shell/                 # generic shell model/update/view, no concrete page imports
  pages/                 # page implementations and simple page helpers
  pages/s3/              # S3 page
  pages/cloudfront/      # CloudFront page
  pageapi/               # shared page contract/state/messages
  workflow/              # shared workflow helpers
```

## Files to modify

- `internal/app/app.go`
  - Stop passing concrete S3/CloudFront services directly into the shell constructor.
  - Build the page registry in `internal/app` and pass it to the shell.

- `internal/app/pages.go` or similar new file
  - Add app-level page registry construction.
  - Instantiate only implemented/real pages: dashboard, S3, and CloudFront.

- `internal/ui/shell/model.go`
  - Change constructors to accept a `[]pages.Page` or equivalent page abstraction instead of S3/CloudFront services.
  - Remove imports of `internal/ui/pages/s3` and `internal/ui/pages/cloudfront`.

- `internal/ui/pages/registry.go`
  - Either remove it, simplify it, or move its registry-building responsibility into `internal/app/pages.go`.
  - Ensure IAM, EC2, and Networking placeholder pages are no longer included in the default page list.

- Tests under `internal/ui/shell/*_test.go`
  - Update shell model construction helpers to pass test page registries directly.

## Reuse

Existing reusable pieces found:

- `internal/ui/pages/page.go`
  - Existing page interface/type used by the shell.

- `internal/ui/pages/dashboard.go`
  - Existing dashboard page constructor.

- `internal/ui/pages/resource.go`
  - Existing placeholder/resource page constructor. After this change it may become unused; if so, remove it and its tests/references rather than keeping dead placeholder code.

- `internal/ui/pages/s3/s3.go`
  - Existing S3 page constructors, including preference-aware constructor.

- `internal/ui/pages/cloudfront/cloudfront.go`
  - Existing CloudFront page constructor.

- `internal/app/app.go`
  - Existing dependency wiring point; should own concrete composition.

## Steps

- [x] Add `internal/app/pages.go` with a function that builds the default app page list from S3 service, CloudFront service, and preference store.
- [x] Move the current page list construction logic from `internal/ui/pages/registry.go` into the new app-level page registry function.
- [x] Remove IAM, EC2, and Networking placeholder pages from the default registry; keep only dashboard, S3, and CloudFront.
- [x] Change `shell.NewModelWithPreferences` so it accepts a prebuilt page registry instead of concrete S3 and CloudFront services.
- [x] Change `shell.NewModel` similarly, or remove/adjust it if it is only a convenience wrapper.
- [x] Update `internal/app/app.go` to call the new app page registry builder and pass the resulting pages into `shell.NewModelWithPreferences`.
- [x] Remove concrete `s3` and `cloudfront` page imports from `internal/ui/shell/model.go`.
- [x] Decide whether `internal/ui/pages/registry.go` should be deleted or kept only for UI-local generic page helpers.
- [x] Remove `internal/ui/pages/resource.go` if no remaining code references `NewResourcePage`.
- [x] Update shell tests to construct models with explicit test page registries.
- [x] Run formatting and tests.

## Verification

- Run all tests:

```bash
go test ./...
```

- Run the app manually:

```bash
go run ./cmd/aws-terminal
```

Manual checks:

- App starts successfully.
- Dashboard still appears.
- Profiles, regions, and page sidebar still render.
- S3 page still opens and receives preferences.
- CloudFront page still opens.
- Placeholder pages like IAM, EC2, and Networking no longer appear.
- Switching focus between Profiles, Regions, Pages, and Page content still works.

## Notes

This plan intentionally avoids a broad directory migration to `features/*`. The current domain/application/infrastructure/ui layering is a good fit for this project. The proposed change gives most of the structural benefit with much lower risk.
