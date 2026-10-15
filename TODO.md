# TODO: Bug Fixes

---

## Critical

### 1. Panic on `ListAccountRoles` API failure ✅ DONE
**File:** `internal/aws.go:16`
**Issue:** The error from `ssoClient.ListAccountRoles()` is silently discarded with `_`. If the API call fails, `roles` is `nil`, and `len(roles.RoleList)` on line 18 causes a nil pointer dereference panic.
**Fix:** Check the error from `ListAccountRoles` before accessing `roles.RoleList`. Use `check(err)` (consistent with the rest of the codebase) to log and fatal gracefully.

---

### 2. Wrong expiration in credential_process JSON output ✅ DONE
**File:** `internal/assume.go:41`
**Issue:** `Expiration: time.Now().Add(1 * time.Hour).Format(time.RFC3339)` uses a hardcoded 1-hour offset instead of the actual credential expiration from the AWS API. The persisted mode (`main.go:221`) correctly uses `time.Unix(*roleCredentials.RoleCredentials.Expiration/1000, 0)`. The credential_process JSON output should use the same.
**Fix:** Replace `time.Now().Add(1 * time.Hour)` with `time.Unix(*roleCredentials.RoleCredentials.Expiration/1000, 0)`.

---

### 3. Headless mode detection is broken ✅ DONE
**File:** `pkg/sso/aws.go:175`
**Issue:** Uses `strings.Contains(strings.Join(os.Args, ","), "--headless")` — a fragile substring match on raw args. The `--headless` CLI flag value is never threaded down to `startDeviceAuthorization`; the function has no access to the parsed CLI context.
**Fix:** Add a `headless bool` parameter to `startDeviceAuthorization`, `registerClient`, and `ProcessClientInformation` (or the call chain). Pass `context.Bool("headless")` from the command actions. Remove the string-matching hack.

---

### 4. `applyForceFlag` unconditionally logs success ✅ DONE
**File:** `cmd/go-aws-sso/main.go:258-271`
**Issue:** Lines 264 and 269 (`zap.S().Infof("Removed temporary access token")` and `zap.S().Infof("Removed temporary lock file")`) are printed unconditionally, even when the preceding `os.Remove` failed and the file was not actually removed.
**Fix:** Move the success log messages inside the `if err == nil` block (or into an `else` branch of the existing error check).

---

## Medium

### 5. `ReadClientInformation` ignores its file parameter
**File:** `pkg/sso/file_system.go:106-114`
**Issue:** The function checks if `file` exists (line 107) but then reads from `ClientInfoFileDestination()` (line 109) instead of the `file` parameter. Currently latent because both callers pass `ClientInfoFileDestination()`, but this is a logic bug.
**Fix:** Replace `os.ReadFile(ClientInfoFileDestination())` with `os.ReadFile(file)`.

---

### 6. Token expiration ignores `ExpiresIn` from the API
**File:** `pkg/sso/aws.go:248`
**Issue:** `retrieveToken` sets expiration to `timer.Now().Add(time.Hour*8 - time.Minute*5)` — a hardcoded 8-hour duration. The `CreateTokenOutput` response has an `ExpiresIn` field (`*int64`, seconds) that should be used instead. If AWS changes the default, cached tokens will expire at the wrong time.
**Fix:** Read `*cto.ExpiresIn` from the `CreateTokenOutput` and compute expiration as `timer.Now().Add(time.Duration(*cto.ExpiresIn) * time.Second - time.Minute*5)`.

---

### 7. TOCTOU race condition on authorization lock
**File:** `pkg/sso/aws.go:107-114, 121-129`
**Issue:** `isAuthorizationFlowLocked()` and `lockAuthorizationFlow()` are called sequentially without atomicity. Two concurrent processes can both pass the `isAuthorizationFlowLocked` check before either calls `lockAuthorizationFlow`, defeating the lock's purpose.
**Fix:** Use `os.OpenFile` with `O_CREATE|O_EXCL` for atomic lock creation, or use a dedicated file locking library (e.g., `flock`). Alternatively, accept the small race window but document it.

---

### 8. `lockAuthorizationFlow` silently fails to write
**File:** `pkg/sso/aws.go:262-273`
**Issue:** If `os.WriteFile` fails (disk full, permission denied), the error is logged but not returned. The caller proceeds as if the lock was acquired. Concurrent auth flows could result.
**Fix:** Return the error or use `check(err)` to fatal (consistent with the rest of the codebase). A failed lock write should stop execution since the lock is the primary concurrency guard.

---

### 9. `BROWSER` env var error is fatal, default browser error is not
**File:** `pkg/sso/aws.go:182-210`
**Issue:** If `BROWSER` is set but points to a non-existent binary, the program calls `zap.S().Fatalf` and exits (line 189). If the default system browser fails (e.g., `xdg-open` not found), only a warning is logged (line 208). The behavior should be consistent.
**Fix:** Change the `BROWSER` error from `Fatalf` to `Error` to match the default path, or change both to be fatal with a clear message advising the user to use `--headless`.

---

### 10. `cfg.SaveTo()` error silently ignored
**File:** `pkg/sso/file_system.go:80`
**Issue:** `cfg.SaveTo(CredentialsFilePath)` returns an error that is discarded. If the credentials file can't be saved (disk full, permissions), the user gets silent failure.
**Fix:** Add `check(err)` after the call.

---

### 11. `os.WriteFile` error silently ignored in `WriteStructToFile`
**File:** `pkg/sso/file_system.go:126`
**Issue:** `_ = os.WriteFile(dest, file, 0600)` — the error is discarded. This writes the OIDC access token cache and last-usage cache. A silent write failure means stale/corrupted cache that manifests as confusing errors later.
**Fix:** Add `check(err)` (or at minimum `zap.S().Error(err)`).

---

### 12. Config file written with executable permissions
**File:** `internal/config.go:87`
**Issue:** `os.WriteFile(filePath, bytes, 0755)` grants `rwxr-xr-x` to a YAML config file containing `start-url` and `region`. No reason for it to be executable.
**Fix:** Change `0755` to `0600` or `0644`.

---

## Low / Code Quality

### 13. Inconsistent `applyForceFlag` ordering between commands
**Files:** `cmd/go-aws-sso/main.go:168-169` vs `main.go:109,124`
The default action calls `InitClients` before `applyForceFlag`. The `refresh` and `assume` commands call `applyForceFlag` first. Behavior is functionally identical but inconsistent.
**Fix:** Move `applyForceFlag` before `InitClients` in the default action to match the other commands.

---

### 14. Duplicate `check()` function
**Files:** `internal/common.go:7-10` and `pkg/sso/file_system.go:128-131`
Identical implementations in two packages.
**Fix:** Remove one copy. If `internal` depends on `pkg/sso`, the one in `pkg/sso` could be exported and reused. Or keep both if circular dependency concerns exist.

---

### 15. `createCredentialsFile` uses bare `O_CREATE`
**File:** `pkg/sso/file_system.go:68`
`os.OpenFile(CredentialsFilePath, os.O_CREATE, 0644)` without `O_WRONLY`/`O_RDWR` is unconventional. The file handle is immediately closed (deferred), and the next operation uses `ini.Load`/`ini.SaveTo`, so it works but is non-idiomatic.
**Fix:** Replace with `os.Create(CredentialsFilePath)`.

---

### 16. `sec.ReflectFrom` error unchecked
**File:** `pkg/sso/file_system.go:89`
The `err` from `sec.ReflectFrom(template)` is discarded. If INI reflection fails, the section is silently empty.
**Fix:** Add `check(err)`.

---

### 17. `os.UserHomeDir()` error silently ignored
**Files:** `pkg/sso/aws.go:91`, `internal/refresh.go:77`, `internal/refresh.go:88`
If `$HOME` is unset, `homeDir` is empty string, producing malformed paths like `/.aws/sso/cache/...`. `pkg/sso/file_system.go:52` correctly calls `check(err)` for the same function.
**Fix:** Add `check(err)` or a fallback to all call sites.

---

### 18. Missing tests for error/recovery paths
Untested scenarios:
- 401 unauthorized → cache removal → retry (`retryWithNewClientCreds`)
- Lock file contention between two processes
- Headless mode (no browser open, verification URL printed)
- Corrupted `access-token.json` → re-registration
- Token expiry → re-auth flow
- `BROWSER` env var behavior
- Missing config file fallback in `readConfigFile`
- `--force` flag combined with missing files

---

### 19. Static region list excludes newer regions
**File:** `pkg/sso/aws.go:30-54`
`AwsRegions` is a hardcoded list. Newer AWS regions (e.g., `ap-southeast-5`, `me-central-1`, `eu-central-2`, `ap-south-2`) are missing. The promptui fuzzy matcher allows typing arbitrary input, but unmatched input defaults to index 0 (`us-east-2`), which silently selects the wrong region.
**Fix:** Update the list with all current regions, or switch to region discovery via the AWS API, or map unmatched free-text input to the typed value rather than index 0.
