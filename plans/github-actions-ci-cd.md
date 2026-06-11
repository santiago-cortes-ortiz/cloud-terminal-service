# Plan: GitHub Actions CI and release builds

## Context

The repository is a Go TUI app (`module aws-terminal`) with entrypoint `cmd/aws-terminal/main.go`. The goal is to add GitHub-hosted CI/CD:
- Run all tests automatically.
- Build release binaries for Windows, Linux, macOS ARM, and macOS Intel only on releases.
- Create/populate a GitHub Release with generated changelog notes and binary artifacts.

Initial findings:
- There is currently no `.github/` workflow configuration.
- Common project commands in `AGENTS.md`: `go test ./...`, `go build ./...`.
- The executable name should likely be `aws-terminal` (`aws-terminal.exe` on Windows).
- `go.mod` currently declares `go 1.26.4`; GitHub Actions setup needs to account for that Go version availability.

## Approach

Use GitHub Actions with separate workflows:
- CI workflow for pushes and pull requests: check formatting, run `go test ./...`, and run `go build ./...`.
- Release workflow triggered by version tag pushes, e.g. `v*`: run tests first, cross-compile binaries using Go environment variables, archive them, create a GitHub Release, auto-generate release notes from commit history/PRs, and upload artifacts to the release.
- Initially use the Go version from `go.mod` (`go 1.26.4`) via `actions/setup-go` and only revisit if GitHub Actions cannot install it.

## Files to modify

Likely files:
- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
- `README.md` if release/tagging instructions should be documented.

## Reuse

Existing commands/patterns to reuse:
- `go test ./...` from `AGENTS.md`.
- `go build ./...` from `AGENTS.md`.
- Build entrypoint: `./cmd/aws-terminal`.
- Module/build metadata from `go.mod`.

## Steps

- [x] Confirm release trigger/versioning preference and changelog source: release on tag push, use GitHub auto-generated release notes, and rely on conventional commits for good changelog content.
- [x] Add GitHub Actions CI workflow using `actions/checkout` and `actions/setup-go` with Go module caching, initially using the Go version from `go.mod`.
- [x] Add release workflow triggered by tag pushes matching `v*`, with a test gate before packaging.
- [x] Cross-compile release assets for:
  - [x] `windows/amd64` -> `.zip` containing `aws-terminal.exe`
  - [x] `linux/amd64` -> `.tar.gz` containing `aws-terminal`
  - [x] `darwin/arm64` -> `.tar.gz` containing `aws-terminal`
  - [x] `darwin/amd64` -> `.tar.gz` containing `aws-terminal`
- [x] Create the GitHub Release from the pushed tag and enable generated release notes/changelog.
- [x] Upload binaries to the GitHub Release with `GITHUB_TOKEN` permissions.
- [x] Optionally document the release process in `README.md`.

## Verification

- Validate workflow YAML syntax locally by inspection.
- Push a branch/PR and verify CI runs tests/builds.
- Create a test tag/release and verify release job creates expected assets and release notes.
- Download each artifact and verify archive contents/naming.
