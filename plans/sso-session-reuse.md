# SSO Session Reuse Plan

## Context

The TUI currently starts the native AWS SSO device authorization flow whenever a user presses `enter` on an SSO profile. This forces a browser/device-code login on each app launch even if AWS already has a valid cached SSO access token for that profile.

Initial code findings:
- `internal/ui/shell/update.go` initiates auth from profile selection and activates the session only after SSO polling completes.
- `internal/application/authentication/service.go` always calls `DeviceFlowAuthenticator.Start` for SSO profiles.
- `internal/infrastructure/awssso/device_flow.go` already writes AWS SDK-compatible SSO token cache files using `ssocreds.StandardCachedTokenFilepath(profile.SSO.CacheKey())`.
- `internal/application/session/service.go` / `internal/infrastructure/awsconfig` resolve the active identity through normal AWS SDK config loading.

## Approach

Add a preflight “cached SSO session” check before starting the device-flow login. The check should use the same AWS SDK SSO token provider/cache location as normal AWS config loading:

1. Resolve the profile’s SSO token cache path with `ssocreds.StandardCachedTokenFilepath(profile.SSO.CacheKey())`.
2. Build an `ssooidc` client for the profile’s `sso_region`.
3. Call `ssocreds.NewSSOTokenProvider(client, cachePath).RetrieveBearerToken(ctx)`.
   - If the cached access token is still valid, this succeeds.
   - If the cached access token is expired but has a refresh token/client registration, the SDK provider refreshes it and rewrites the cache.
   - If the cache is missing, invalid, expired without refresh data, or refresh fails, return `false` without treating it as a fatal app error.
4. In the TUI profile-selection flow, for SSO profiles call this check first. If it returns reusable, skip browser/device-code auth and directly call the existing `ActivateProfile` command. If it returns not reusable, start the current native device authorization flow unchanged.

This keeps the AWS SDK as the source of truth for SSO cache format and refresh behavior while preserving the current native login fallback.

## Files to modify

- `internal/application/authentication/ports.go` — extend the authentication port with cached-session reuse capability.
- `internal/application/authentication/service.go` — add `HasReusableSSOSession(ctx, profile) (bool, error)` validation and input checks.
- `internal/infrastructure/awssso/device_flow.go` or new `internal/infrastructure/awssso/session_cache.go` — implement the cached token lookup/refresh using AWS SDK `ssocreds.NewSSOTokenProvider`.
- `internal/ui/shell/model.go` — extend `AuthenticationService`, add cancellation slot for cache-check command if needed.
- `internal/ui/shell/messages.go` — add a message for cached-session check completion.
- `internal/ui/shell/update.go` — branch SSO profile activation through cache reuse before starting device flow.
- `internal/infrastructure/awssso/device_flow_test.go` or new `session_cache_test.go` — token-cache validation tests.
- `internal/application/authentication/service_test.go` — service-level branching/input tests.
- `internal/ui/shell/update_test.go` — UI branch tests for reusable vs non-reusable SSO sessions.
- `README.md` — document that SSO sessions are reused/refreshed until AWS cache refresh is no longer possible.

## Reuse

- Reuse `profile.UsesSSO()` / `profile.SSO.CacheKey()` from `internal/domain/profile/profile.go`.
- Reuse `newOIDCClient(ctx, region)` from `internal/infrastructure/awssso/device_flow.go` for token refresh calls.
- Reuse AWS SDK SSO cache path resolution via `ssocreds.StandardCachedTokenFilepath`, already used in `internal/infrastructure/awssso/device_flow.go`.
- Reuse `ssocreds.NewSSOTokenProvider(...).RetrieveBearerToken(ctx)` from AWS SDK to validate unexpired tokens and refresh expired-but-refreshable tokens.
- Reuse `ActivateProfile` / identity resolution in `internal/application/session/service.go` after cache validation.
- Reuse the existing `startSSOLoginCmd`, `pollSSOLoginCmd`, and `activateProfileCmd` paths in `internal/ui/shell/update.go`.

## Steps

- [x] Extend `authentication.DeviceFlowAuthenticator` with `HasReusableSession(ctx, profile) (bool, error)` or introduce a sibling `SSOSessionCache` port. Prefer adding the method to the existing injected auth adapter to keep wiring simple.
- [x] Add `Service.HasReusableSSOSession(ctx, profile)` in `internal/application/authentication/service.go`:
  - validate profile name and SSO config like `StartSSOLogin` does;
  - call the infrastructure port;
  - return `false, nil` for normal cache-miss/not-reusable cases and return errors only for invalid input or unexpected validation failures worth surfacing.
- [x] Implement `OIDCDeviceFlowAuthenticator.HasReusableSession`:
  - compute cache path with `ssocreds.StandardCachedTokenFilepath(profile.SSO.CacheKey())`;
  - construct OIDC client with `newOIDCClient(ctx, profile.SSO.Region)`;
  - call `ssocreds.NewSSOTokenProvider(client, cachePath).RetrieveBearerToken(ctx)`;
  - return `true` if the returned bearer token has a non-empty value and is not already expired;
  - return `false` for missing cache/expired-unrefreshable/refresh-denied errors so the UI falls back to login.
- [x] Update shell interfaces and messages:
  - add `HasReusableSSOSession` to `AuthenticationService` in `internal/ui/shell/model.go`;
  - add `ssoSessionCheckedMsg { profile domainprofile.Profile; reusable bool; err error }` in `messages.go`;
  - add cancellation handling for the new async command.
- [x] Change `activateSelectedProfile` for SSO profiles:
  - set status to “Checking cached AWS SSO session…”;
  - run `checkSSOSessionCmd(profile)` instead of immediately calling `startSSOLoginCmd`.
- [x] Handle `ssoSessionCheckedMsg` in `Update`:
  - if canceled, clear busy state;
  - if reusable, set status to “Reusing cached AWS SSO session…” and call `activateProfileCmd(profile.Name)`;
  - if not reusable or cache check errors, set status to “Starting native AWS SSO login…” and call `startSSOLoginCmd(profile)`.
- [x] Keep the existing post-device-flow behavior unchanged: after polling completes, call `activateProfileCmd(profileName)`.
- [x] Add tests for infrastructure cache behavior, application validation, and shell branch behavior.
- [x] Update README auth wording from “Press enter ... authenticate” to clarify that `enter` activates with a cached/refreshed SSO session when available and only opens login when needed.

## Verification

- Run `go test ./...`.
- Unit-test cache cases:
  - unexpired token returns reusable;
  - expired token with refresh fields uses `CreateToken` and returns reusable;
  - missing/malformed/expired-unrefreshable cache returns not reusable.
- Unit-test shell behavior:
  - SSO + reusable cache calls activation and does not start SSO login;
  - SSO + non-reusable cache starts device login;
  - non-SSO profiles still activate directly.
- Manual: log in once, quit app, relaunch, select same SSO profile, verify no browser/device-code prompt appears and caller identity resolves.
- Manual: with an expired but refreshable AWS SSO cache, relaunch and select the profile, verify no browser prompt and cache `expiresAt` updates.
- Manual: delete `~/.aws/sso/cache/<profile-or-session-hash>.json`, select SSO profile, verify existing browser/device-flow login still works.
