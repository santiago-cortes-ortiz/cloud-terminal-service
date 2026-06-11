# aws-terminal

Terminal UI for creating and managing AWS resources with Go and Bubble Tea.

## Stack

- Go
- Bubble Tea
- Bubbles
- Lip Gloss
- AWS SDK for Go v2

## Structure

```text
cmd/aws-terminal                     # application entrypoint
internal/app                         # dependency wiring, app page registry, and program bootstrap
internal/domain/auth                 # native SSO device-flow domain types
internal/domain/cloudfront           # CloudFront distribution/invalidation entities
internal/domain/ecr                  # ECR repositories/images and Docker push entities
internal/domain/profile              # profile entities and authentication mode
internal/domain/region               # AWS region catalog/types
internal/domain/s3                   # S3 bucket and sync plan entities
internal/domain/session              # active AWS session/account state
internal/application/authentication  # SSO authentication use cases
internal/application/cloudfront      # CloudFront listing/invalidation use cases
internal/application/ecr             # ECR repository/image and Docker push use cases
internal/application/s3              # S3 listing, source inspection, sync planning, sync execution
internal/application/session         # profile/session use cases and ports
internal/infrastructure/awsclients    # shared AWS SDK client factory/cache
internal/infrastructure/awscloudfront # AWS SDK CloudFront adapter
internal/infrastructure/awsconfig     # shared config parsing and STS identity resolution
internal/infrastructure/awsecr       # AWS SDK private ECR adapter
internal/infrastructure/awss3        # AWS SDK S3 adapters
internal/infrastructure/localdocker  # Docker Engine API adapter
internal/infrastructure/awssso       # native AWS SSO OIDC device-flow implementation
internal/ui/pageapi                  # shared Page contract and page messages/state
internal/ui/workflow                 # reusable page workflow helpers
internal/ui/shell                    # Bubble Tea shell model, update loop, keymap
internal/ui/components               # reusable layout components (header, footer, sidebar)
internal/ui/pages                    # page implementations and page type re-exports
internal/ui/pages/s3                 # S3 workflow page
internal/ui/pages/cloudfront         # CloudFront workflow page
internal/ui/pages/ecr                # ECR workflow page
internal/ui/styles                   # shared Lip Gloss theme and rendering helpers
```

## Architecture

The project is organized to stay scalable as more AWS workflows are added:

- **domain**: core types with no UI or AWS SDK knowledge
- **application**: use cases and small interfaces
- **infrastructure**: AWS SDK and native SSO/OIDC adapters
- **ui**: Bubble Tea shell, reusable components, page contracts, and page rendering

Concrete page composition lives in `internal/app/pages.go`, so `internal/ui/shell` remains page-agnostic. The current app page registry includes Dashboard, S3, CloudFront, and ECR.

This keeps the TUI separate from authentication, AWS config loading, and future resource creation services.

## Run

```bash
go run ./cmd/aws-terminal
```

## AWS profiles and SSO

- Profiles are loaded from your local AWS shared config/credentials files.
- In the TUI, switch focus between **Profiles**, **Regions**, and **Pages** with `tab`.
- Use `shift+tab` to move focus back.
- Use `↑/↓` to move within the focused section.
- Press `enter` on a region to make it the active region for the session and upcoming workflows.
- The left sidebar uses fixed-height `bubbles/list` panes for Profiles, Regions, and Pages.
- Press `enter` on a profile to activate it in the selected region. For AWS SSO profiles, the app first reuses or refreshes the cached SSO session when available and only opens the browser/device login when a new sign-in is needed.
- Press `enter` on **Pages** to move into the active page workflow.
- For AWS SSO profiles, the app uses the native AWS SSO OIDC device flow directly from Go when the cached session cannot be reused.
- The TUI stays visible while SSO cache checks or authentication runs.
- The selected region is shown in the sidebar, footer, and dashboard.
- The dashboard shows the verification URL, one-time user code, and browser-open status.
- After approval completes, the app writes the AWS SSO token cache and resolves the active caller identity. Future launches reuse that cache until AWS expires it or it can no longer be refreshed.
- Your AWS SSO profiles still need to be configured in `~/.aws/config`.
- The AWS CLI is no longer required for the login flow itself.

## S3 sync workflow

The first interactive resource workflow is the **S3 Buckets** page.

### Current capabilities

- List buckets available to the active authenticated profile.
- Choose a destination bucket from inside the page.
- Browse your local machine with the Bubbles file picker.
- Select either a **file** or a **folder** as the source.
- Enter an optional S3 prefix.
- Build a sync review step before execution.
- Explicitly toggle delete behavior on/off before running the sync.
- Execute the sync asynchronously and show in-page status/progress updates.
- After a successful sync, optionally jump into CloudFront invalidation.

### Page flow

1. Authenticate a profile and pick a region.
2. Open the **S3 Buckets** page from the sidebar.
3. Press `enter` while the Pages pane is focused to move into the active page.
4. Select the destination bucket.
5. Browse for a local source path.
6. Enter an optional destination prefix.
7. Review the sync plan.
8. Toggle delete behavior if needed.
9. Confirm and run the sync.
10. After success, press `i` if you want to continue into CloudFront invalidation.

### S3 page keys

#### Bucket selection

- `↑/k` move up
- `↓/j` move down
- `enter` choose bucket
- `tab` or `shift+tab` return focus to Pages

#### File picker

- `right` or `l` open directory
- `enter` or `space` select the highlighted file or folder
- `backspace` go to parent directory
- `b` go back to bucket selection

#### Prefix / review / sync

- `enter` continue / confirm
- `space` toggle delete on the review screen
- `b` go back one step
- `i` open CloudFront after a successful sync prompt
- `tab` or `shift+tab` return focus to Pages

## ECR private repository workflow

The **ECR** page supports private ECR repositories only for now.

### Current capabilities

- List and search private ECR repositories for the active authenticated profile/region.
- Create a private ECR repository from the page when needed.
- Select a repository and view existing image tags/digests.
- List local Docker images through the Docker Engine API.
- Manually type a local Docker image reference if discovery is unavailable or incomplete.
- Edit the destination tag before pushing.
- Retrieve ECR authorization, tag the local image for the selected repository, push it, and show progress.

### ECR page keys

- `↑/k` and `↓/j` move through repository/local image lists.
- `enter` select/continue/confirm.
- `ctrl+f` focus repository search; `esc` leaves search.
- `n` create a repository from the repository step.
- `r` refresh repositories, repository images, or local Docker images depending on the current step.
- `b` or `esc` go back; during push it cancels waiting for the running push context.
- `tab` or `shift+tab` return focus to Pages.

### ECR manual verification

1. Run `go run ./cmd/aws-terminal`.
2. Select the AWS profile and region that should own the private ECR repository.
3. Open the **ECR** page.
4. Press `ctrl+f` to search for an existing repository or press `n` to create a test repository.
5. Confirm repository images are displayed.
6. Ensure Docker is running locally.
7. Select a discovered local image, or type a local image reference manually.
8. Edit the destination tag.
9. Review and confirm the push.
10. Verify the image appears in private ECR, for example:

    ```bash
    aws ecr describe-images --repository-name <repo> --profile <profile> --region <region>
    ```

## CloudFront invalidation workflow

- Open the **CloudFront** page directly from the sidebar, or press `i` after a successful S3 sync.
- Select a distribution.
- Enter one or more invalidation paths such as `/*` or `/assets/* /index.html`.
- Press `enter` to create the invalidation.
- Press `c` to copy the equivalent AWS CLI command to your clipboard.

## Sync behavior notes

- The delete option is **never implicit**. It must be explicitly enabled on the confirmation screen.
- When the selected source is a **single file**, delete is automatically disabled.
- The current sync planning compares local files to remote objects by destination key and size.
- Folder sync preserves relative paths under the chosen prefix.
- An empty prefix means the bucket root.

## Expected manual verification flow

Use real configured profiles and buckets on your own machine.

### Example targets

- `pre` profile syncing into `s3://wavy-pre/`
- `prod` profile syncing into `s3://wavy-prod/`

### Manual verification checklist

1. Run the app:

   ```bash
   go run ./cmd/aws-terminal
   ```

2. Select the region that matches the bucket you want to use.
3. Authenticate either `pre`, `prod`, or another configured profile.
4. Open the **S3 Buckets** page.
5. Confirm the expected bucket appears in the list.
6. Choose the bucket.
7. Use the file picker to select a folder such as:

   ```text
   dist/wavycat-frontend-v3/browser
   ```

8. Leave the prefix empty to target bucket root, or enter a prefix such as:

   ```text
   browser
   ```

9. Review the plan.
10. Confirm delete is **off** first and run a safe sync.
11. Validate objects in AWS or with the AWS CLI, for example:

    ```bash
    aws s3 ls s3://wavy-pre/ --profile pre
    aws s3 ls s3://wavy-prod/ --profile prod
    ```

12. Run the workflow again with delete **on** only after reviewing the listed deletions.
13. Compare the behavior with your existing CLI workflow:

    ```bash
    aws s3 sync dist/wavycat-frontend-v3/browser s3://wavy-pre/ --delete --profile pre
    aws s3 sync dist/wavycat-frontend-v3/browser s3://wavy-prod/ --delete --profile prod
    ```

## Development status

- The S3, CloudFront, and ECR pages are implemented and wired through the app page registry.
- The app builds successfully with `go build ./...`.
- Real AWS verification still depends on local profiles, buckets, and manual execution in your environment.
