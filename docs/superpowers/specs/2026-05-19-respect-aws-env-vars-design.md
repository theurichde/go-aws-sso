# Respect AWS Config/Credentials Env Vars

Issue: [#195](https://github.com/theurichde/go-aws-sso/issues/195)

## Problem

go-aws-sso hardcodes `~/.aws/credentials`, `~/.aws/sso/cache/access-token.json`, and `~/.aws/sso/cache/last-usage.json` — ignoring the standard AWS environment variables `AWS_SHARED_CREDENTIALS_FILE` and `AWS_CONFIG_FILE`.

Setting these env vars has no effect on where files are read or written, breaking workflows where users redirect their AWS file locations.

## Design Decisions

- **Respect only `AWS_SHARED_CREDENTIALS_FILE`**, not `AWS_CONFIG_FILE`. go-aws-sso doesn't use `~/.aws/config`, so `AWS_CONFIG_FILE` would only affect SSO cache derivation — thin justification for the added complexity. A single variable for both credentials and SSO cache covers the actual use case and avoids ambiguity (what happens when both point to different directories?).
- **Derive SSO cache from credentials directory**, matching AWS CLI behavior. If `AWS_SHARED_CREDENTIALS_FILE=/custom/path/credentials`, the SSO cache goes to `/custom/path/sso/cache/`.
- **Centralize path resolution** in a new `pkg/sso/paths.go` file (same `sso` package) instead of scattering env var checks across call sites.
- **Fix `ReadClientInformation` bug** as part of this change: the function currently ignores its `file` parameter and always reads from the hardcoded path.

## Architecture

### New file: `pkg/sso/paths.go`

Five package-level functions, all in the `sso` package:

| Function | Behavior |
|---|---|
| `AwsDir()` | If `AWS_SHARED_CREDENTIALS_FILE` is set and non-empty, returns `filepath.Dir(envValue)`. Otherwise returns `~/.aws`. |
| `GetCredentialsFilePath()` | Moved from `file_system.go`. If `AWS_SHARED_CREDENTIALS_FILE` is set, returns the env var value directly. Otherwise returns `AwsDir() + "/credentials"`. |
| `SsoCacheDir()` | `AwsDir() + "/sso/cache"` |
| `ClientInfoFilePath()` | Renamed from `ClientInfoFileDestination()`. Returns `SsoCacheDir() + "/access-token.json"`. |
| `LastUsageFilePath()` | Returns `SsoCacheDir() + "/last-usage.json"`. |

The functions are intentionally simple to avoid adding a `pathConfig` struct or receiver — there's no state to manage, and env vars are resolved at call time (set before process start for a CLI tool).

Edge case: if `AWS_SHARED_CREDENTIALS_FILE=mycreds` (filename only, no directory), `filepath.Dir("mycreds")` returns `"."`, so SSO cache goes to `./sso/cache/`. This matches AWS CLI behavior.

### Modified files

**`pkg/sso/file_system.go`**
- Remove `GetCredentialsFilePath()` function (moved to `paths.go`).
- Remove `ClientInfoFileDestination()` function (renamed to `ClientInfoFilePath()`).
- The package var `CredentialsFilePath` still initializes via `GetCredentialsFilePath()` — now from `paths.go`.
- Fix `ReadClientInformation`: use the `file` parameter for `os.ReadFile` instead of the hardcoded `ClientInfoFileDestination()`.

**`pkg/sso/aws.go`**
- Delete `ClientInfoFileDestination()` function (lines 90-93).
- Update all callers to `ClientInfoFilePath()`:
  - Line 111: `ReadClientInformation(ClientInfoFilePath())`
  - Line 119: `WriteStructToFile(clientInfoPointer, ClientInfoFilePath())`
  - Line 139: `WriteStructToFile(clientInfoPointer, ClientInfoFilePath())`

**`internal/refresh.go`**
- `SaveUsageInformation()`: replace `homeDir + "/.aws/sso/cache/last-usage.json"` with `LastUsageFilePath()`. Remove `os.UserHomeDir()` call.
- `readUsageInformation()`: same replacement.
- `retryWithNewClientCreds()`: update `os.Remove(ClientInfoFileDestination())` to `os.Remove(ClientInfoFilePath())`.

**`cmd/go-aws-sso/main.go`**
- `applyForceFlag()`: update two `os.Remove(ClientInfoFileDestination())` calls to `os.Remove(ClientInfoFilePath())`.

### Test file: `pkg/sso/paths_test.go`

Test cases:

| Scenario | Input | Expected |
|---|---|---|
| Env var set to custom path with directory | `AWS_SHARED_CREDENTIALS_FILE=/tmp/aws/credentials` | CredentialsFilePath → `/tmp/aws/credentials`, SsoCacheDir → `/tmp/aws/sso/cache` |
| Env var unset | (default) | All paths default to `~/.aws/...` |
| Env var set but empty | `AWS_SHARED_CREDENTIALS_FILE=` | Fall back to defaults (empty → treated as unset) |
| Env var set to filename only | `AWS_SHARED_CREDENTIALS_FILE=mycreds` | CredentialsFilePath → `mycreds`, SsoCacheDir → `sso/cache` (i.e., relative to CWD) |

Existing tests in `file_system_test.go`, `aws_test.go`, `assume_test.go`, and `main_test.go` should continue to pass since they don't set `AWS_SHARED_CREDENTIALS_FILE` and defaults are unchanged. Tests that modify `CredentialsFilePath` (e.g., `CredentialsFilePath = temp.Name()`) are unaffected — the var still exists.

### What does NOT change

- `~/.config/go-aws-sso/config.yml` — go-aws-sso's own config file, uses `os.UserConfigDir()`, unrelated to AWS env vars.
- The `--persist`, `--profile`, `--headless` flags.
- The `credential_process` command template — still references the same exe path.
- The authorization flow lock file (`/tmp/go-aws-sso.lock`) — not affected.

## Error handling

- `os.UserHomeDir()` failure: existing `check()` call panics (unchanged from current behavior, unrecoverable).
- Empty `AWS_SHARED_CREDENTIALS_FILE`: treated as unset, falls back to defaults.
- `ReadClientInformation`'s `check(err)` on JSON unmarshal: unchanged.
