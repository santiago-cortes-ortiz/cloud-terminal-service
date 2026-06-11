# ECR Workflow Plan

## Context

Add Amazon ECR as the next AWS service in `aws-terminal`. The intended workflow is to authenticate with an AWS profile/region, search/select ECR repositories, view repository images, and push a selected local Docker image to the selected ECR repository.

Initial codebase findings:
- The app is a Go Bubble Tea TUI with clean layers: `domain`, `application`, `infrastructure`, `ui`, and app wiring in `internal/app`.
- Existing resource pages include S3 and CloudFront, registered in `internal/app/pages.go` and wired in `internal/app/app.go`.
- AWS SDK clients are centralized in `internal/infrastructure/awsclients/factory.go`; currently S3, CloudFront, and STS clients are cached there.
- New resource workflows should follow the AGENTS.md pattern: domain types, application service/ports, infrastructure AWS adapter, page package, app wiring, and tests.

## Approach

Recommended approach: implement ECR as a staged TUI workflow that reuses the existing page/workflow architecture.

Confirmed decisions:
- Private ECR only for v1; do not include ECR Public APIs or `public.ecr.aws` workflows yet.
- Support both local Docker image discovery and manual image reference entry.
- Use the Docker Engine API directly from Go instead of shelling out to `docker`.
- Default the destination tag from the local image tag, but allow editing before push.
- Allow creating a private ECR repository from the workflow when search finds no matching existing repository or when the user explicitly chooses create.

Proposed stages:
1. Load ECR repositories for the active profile/region and provide an in-page search/filter input.
2. Select an existing repository or choose a create-repository action.
3. If creating, enter repository name, create it through ECR, then continue with that repository selected.
4. Load and view repository images/tags/digests for the selected repository.
5. List local Docker images from the Docker daemon and also allow manual image reference entry.
6. Edit/confirm the destination tag and review push details: source local image, destination ECR URI/tag, profile, region, repository.
7. Retrieve ECR authorization, authenticate the Docker Engine client to the ECR registry, tag if needed, push, and stream progress/status back into the page.

## Files to modify

Likely additions/changes:
- `go.mod`, `go.sum` — add `github.com/aws/aws-sdk-go-v2/service/ecr` and a Docker Engine client module such as `github.com/docker/docker/client`.
- `internal/domain/ecr/types.go`
- `internal/application/ecr/ports.go`
- `internal/application/ecr/service.go`
- `internal/infrastructure/awsecr/service.go`
- `internal/infrastructure/localdocker/service.go`
- `internal/infrastructure/awsclients/factory.go`
- `internal/ui/pages/ecr/*`
- `internal/app/app.go`
- `internal/app/pages.go`
- Tests under `internal/application/ecr`, `internal/infrastructure/awsecr`, and `internal/ui/pages/ecr`.
- `README.md`, possibly `AGENTS.md` after implementation.

## Reuse

Existing patterns/utilities to reuse:
- App registration/wiring: `internal/app/app.go`, `internal/app/pages.go`.
- AWS client cache/config loading: `internal/infrastructure/awsclients/factory.go`.
- AWS adapter shape and SDK mapping: `internal/infrastructure/awscloudfront/service.go`.
- Page workflow conventions: `internal/ui/pages/s3/*`, `internal/ui/pages/cloudfront/*`.
- S3 async progress/event channel pattern for long operations: `internal/ui/pages/s3/s3_commands.go`.
- Shared page API: `internal/ui/pageapi/page.go`.
- Workflow helpers: `internal/ui/workflow/workflow.go`.
- Existing async owned message pattern from S3/CloudFront page commands.

## Scope notes

- Private ECR is the standard per-AWS-account/per-region service used for application container images, with repository URIs like `<account>.dkr.ecr.<region>.amazonaws.com/my-repo`.
- ECR Public is intentionally out of scope for v1 because it uses different AWS APIs and `public.ecr.aws/...` URI semantics.
- Docker must be running and reachable by the Docker Engine client; if it is not, the page should still allow repository browsing and show an actionable local Docker error only in the local-image/push stages.

## Steps

- [x] Add ECR domain types for repositories, repository images, local Docker images, create-repository input, push plans, progress, and results.
- [x] Add ECR application service with ports for ECR repository/image operations and local Docker operations. Validate profile/region/repository/image/tag inputs here, not in the UI only.
- [x] Add AWS ECR SDK adapter for `DescribeRepositories`, `CreateRepository`, `DescribeImages`, and `GetAuthorizationToken`; extend the AWS client factory with cached private ECR clients.
- [x] Add local Docker Engine adapter for listing local images, tagging source images for ECR, pushing images, and decoding push progress streams.
- [x] Add Bubble Tea ECR page with repository search, optional create repository, repository image viewing, local image selection/manual entry, destination tag editing, review, push stages, owned async messages, cancellation, and `PageStatus`.
- [x] Wire ECR service/page into app startup and sidebar page registry.
- [x] Add unit tests for service validation/sorting/planning and page state transitions.
- [x] Update documentation with ECR workflow keys and manual verification.

## Verification

- Run `go test ./...`.
- Run `go build ./...`.
- Manual flow:
  1. `go run ./cmd/aws-terminal`.
  2. Select a profile and region with existing ECR repositories.
  3. Open the ECR page.
  4. Search/select a repository, and separately verify creating a test repository from the page.
  5. Confirm images/tags are displayed.
  6. Select a local Docker image from discovered images, then repeat with manual image reference entry.
  7. Edit and review destination ECR URI/tag.
  8. Push and confirm the image appears in ECR.
