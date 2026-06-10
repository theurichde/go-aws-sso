# Wrap zap Logger with Interface (remove Fatal from library code)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace direct `zap.S().Fatal()` calls in library code with a logger interface abstraction, so tests can swap Fatal for a panic/recording variant instead of process exit. Consolidate 3 duplicate `check()` helpers.

**Architecture:** Introduce a `pkg/logger` package with a `Logger` interface and three implementations (`ZapLogger`, `NoopLogger`, `TestLogger`). A package-level `L` variable holds the active logger. All code calls `logger.L.XXX()` instead of `zap.S().XXX()`. `main.go` initializes the logger at startup. Tests set a `TestLogger` where `Fatal` panics (recoverable) instead of calling `os.Exit(1)`.

**Tech Stack:** Go, go.uber.org/zap

**Files touched:** `pkg/logger/` (new), `pkg/sso/aws.go`, `pkg/sso/file_system.go`, `pkg/sso/config.go`, `internal/common.go`, `internal/aws.go`, `internal/assume.go`, `internal/refresh.go`, `internal/config.go`, `internal/prompt.go`, `cmd/go-aws-sso/main.go`, `pkg/sso/aws_test.go`, `pkg/sso/file_system_test.go`, `internal/assume_test.go`, `internal/config_test.go`, `cmd/go-aws-sso/main_test.go`

**Current Fatal/Panicf call sites (9 total):**
| # | File | Line | Call |
|---|------|------|------|
| 1 | `pkg/sso/aws.go` | 108 | `zap.S().Fatal(lockedAuthFlowMsg)` |
| 2 | `pkg/sso/aws.go` | 123 | `zap.S().Fatal(lockedAuthFlowMsg)` |
| 3 | `pkg/sso/aws.go` | 188 | `zap.S().Fatalf("error while opening browser: %s", err)` |
| 4 | `pkg/sso/aws.go` | 242 | `zap.S().Fatal(err)` |
| 5 | `pkg/sso/file_system.go` | 101 | `zap.S().Panicf("Could not determine if file or folder...")` |
| 6 | `pkg/sso/file_system.go` | 130 | `zap.S().Fatalf("Something went wrong: %q", err)` (check helper) |
| 7 | `internal/common.go` | 9 | `zap.S().Fatalf("Something went wrong: %q", err)` (check helper) |
| 8 | `cmd/go-aws-sso/main.go` | 181 | `zap.S().Fatal(err)` |
| 9 | `cmd/go-aws-sso/main.go` | 247 | `zap.S().Fatalf("Something went wrong: %q", err)` (check helper) |

**Duplicate `check()` helpers (3):** `internal/common.go:7-10`, `cmd/go-aws-sso/main.go:245-248`, `pkg/sso/file_system.go:128-131`
Each one does: `if err != nil { zap.S().Fatalf("Something went wrong: %q", err) }`
Plan: consolidate into a single `check()` in `internal` that accepts a logger, OR just use `logger.L.Fatalf(...)` inline.

---

### Task 1: Create `pkg/logger` package (interface + implementations)

**Files:**
- Create: `pkg/logger/logger.go`
- Create: `pkg/logger/logger_test.go`

- [ ] **Step 1: Write `pkg/logger/logger.go`**

```go
// Package logger provides a logger abstraction over zap.
// The package-level L variable is used throughout the application.
// Tests can replace it with a TestLogger to avoid os.Exit on Fatal.
package logger

import (
	"fmt"

	"go.uber.org/zap"
)

// Logger defines the logging interface used throughout the application.
type Logger interface {
	Debug(args ...interface{})
	Debugf(template string, args ...interface{})
	Info(args ...interface{})
	Infof(template string, args ...interface{})
	Warn(args ...interface{})
	Warnf(template string, args ...interface{})
	Error(args ...interface{})
	Errorf(template string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(template string, args ...interface{})
}

// L is the application-wide logger. Set via SetLogger at startup.
// Defaults to NoopLogger (silent) until initialized.
var L Logger = &NoopLogger{}

// SetLogger replaces the global application logger.
func SetLogger(l Logger) {
	L = l
}

// --- ZapLogger: delegates to zap's global sugared logger ---

// ZapLogger implements Logger by wrapping zap.S().
type ZapLogger struct{}

func (z *ZapLogger) Debug(args ...interface{})                    { zap.S().Debug(args...) }
func (z *ZapLogger) Debugf(template string, args ...interface{})  { zap.S().Debugf(template, args...) }
func (z *ZapLogger) Info(args ...interface{})                     { zap.S().Info(args...) }
func (z *ZapLogger) Infof(template string, args ...interface{})   { zap.S().Infof(template, args...) }
func (z *ZapLogger) Warn(args ...interface{})                     { zap.S().Warn(args...) }
func (z *ZapLogger) Warnf(template string, args ...interface{})   { zap.S().Warnf(template, args...) }
func (z *ZapLogger) Error(args ...interface{})                    { zap.S().Error(args...) }
func (z *ZapLogger) Errorf(template string, args ...interface{})  { zap.S().Errorf(template, args...) }
func (z *ZapLogger) Fatal(args ...interface{})                    { zap.S().Fatal(args...) }
func (z *ZapLogger) Fatalf(template string, args ...interface{})  { zap.S().Fatalf(template, args...) }

// --- NoopLogger: discards all messages (used for --quiet) ---

// NoopLogger implements Logger by discarding all log messages.
type NoopLogger struct{}

func (n *NoopLogger) Debug(args ...interface{})                   {}
func (n *NoopLogger) Debugf(template string, args ...interface{}) {}
func (n *NoopLogger) Info(args ...interface{})                    {}
func (n *NoopLogger) Infof(template string, args ...interface{})  {}
func (n *NoopLogger) Warn(args ...interface{})                    {}
func (n *NoopLogger) Warnf(template string, args ...interface{})  {}
func (n *NoopLogger) Error(args ...interface{})                   {}
func (n *NoopLogger) Errorf(template string, args ...interface{}) {}
func (n *NoopLogger) Fatal(args ...interface{})                   {}
func (n *NoopLogger) Fatalf(template string, args ...interface{}) {}

// --- TestLogger: panics on Fatal so tests can recover ---

// TestLogger implements Logger and panics on Fatal calls instead of
// calling os.Exit. Other log levels output via fmt for test visibility.
type TestLogger struct{}

func (t *TestLogger) Debug(args ...interface{})                   {}
func (t *TestLogger) Debugf(template string, args ...interface{}) {}
func (t *TestLogger) Info(args ...interface{})                    {}
func (t *TestLogger) Infof(template string, args ...interface{})  {}
func (t *TestLogger) Warn(args ...interface{})                    {}
func (t *TestLogger) Warnf(template string, args ...interface{})  {}
func (t *TestLogger) Error(args ...interface{}) {
	fmt.Println("ERROR:", fmt.Sprint(args...))
}
func (t *TestLogger) Errorf(template string, args ...interface{}) {
	fmt.Printf("ERROR: "+template+"\n", args...)
}
func (t *TestLogger) Fatal(args ...interface{}) {
	panic("FATAL: " + fmt.Sprint(args...))
}
func (t *TestLogger) Fatalf(template string, args ...interface{}) {
	panic(fmt.Sprintf("FATAL: "+template, args...))
}
```

- [ ] **Step 2: Write `pkg/logger/logger_test.go`**

```go
package logger

import (
	"go.uber.org/zap"
	"testing"
)

func TestZapLogger_InterfaceSatisfaction(t *testing.T) {
	var l Logger = &ZapLogger{}
	_ = l
}

func TestNoopLogger_InterfaceSatisfaction(t *testing.T) {
	var l Logger = &NoopLogger{}
	_ = l
}

func TestTestLogger_InterfaceSatisfaction(t *testing.T) {
	var l Logger = &TestLogger{}
	_ = l
}

func TestTestLogger_FatalPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic from TestLogger.Fatal, got none")
		}
	}()
	l := &TestLogger{}
	l.Fatal("test fatal")
}

func TestTestLogger_FatalfPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic from TestLogger.Fatalf, got none")
		}
	}()
	l := &TestLogger{}
	l.Fatalf("test fatal %s", "arg")
}

func TestSetLogger(t *testing.T) {
	original := L
	defer SetLogger(original)

	SetLogger(&TestLogger{})
	if _, ok := L.(*TestLogger); !ok {
		t.Errorf("expected L to be *TestLogger, got %T", L)
	}
}

func TestDefaultLoggerIsNoop(t *testing.T) {
	original := L
	defer SetLogger(original)

	SetLogger(&NoopLogger{})
	if _, ok := L.(*NoopLogger); !ok {
		t.Errorf("expected default L to be *NoopLogger, got %T", L)
	}
}

func TestNoopLogger_AllMethodsNoPanic(t *testing.T) {
	n := &NoopLogger{}
	n.Debug("x")
	n.Debugf("x %s", "y")
	n.Info("x")
	n.Infof("x %s", "y")
	n.Warn("x")
	n.Warnf("x %s", "y")
	n.Error("x")
	n.Errorf("x %s", "y")
	n.Fatal("x")
	n.Fatalf("x %s", "y")
	// if we get here, no panic, no exit — success
}

func TestSetLogger_Integration(t *testing.T) {
	// Verify the global L variable works as expected
	oldL := L
	defer func() { L = oldL }()

	SetLogger(&TestLogger{})
	if L == nil {
		t.Error("L should not be nil after SetLogger")
	}
}

func TestZapLogger_CallsDontPanic(t *testing.T) {
	// Must replace zap globals with a nop logger so zap.S() works
	zap.ReplaceGlobals(zap.NewNop())
	defer zap.ReplaceGlobals(zap.NewNop())

	z := &ZapLogger{}
	z.Debug("test")
	z.Debugf("test %s", "arg")
	z.Info("test")
	z.Infof("test %s", "arg")
	z.Warn("test")
	z.Warnf("test %s", "arg")
	z.Error("test")
	z.Errorf("test %s", "arg")
	// Don't test Fatal here — it calls os.Exit
}
```

- [ ] **Step 3: Run logger tests to verify they pass**

```bash
go test ./pkg/logger/ -v
```

Expected: all tests pass (9 tests, including interface satisfaction, panic recovery, and no-ops).

- [ ] **Step 4: Commit logger package**

```bash
git add pkg/logger/logger.go pkg/logger/logger_test.go
git commit -m "feat(logger): add Logger interface with Zap, Noop, and Test implementations"
```

---

### Task 2: Replace zap.S() in `pkg/sso/` with `logger.L`

**Files:**
- Modify: `pkg/sso/aws.go`
- Modify: `pkg/sso/file_system.go`
- Modify: `pkg/sso/config.go`

- [ ] **Step 1: Replace in `pkg/sso/aws.go`**

Replace import `"go.uber.org/zap"` with `logger "github.com/theurichde/go-aws-sso/pkg/logger"`.

Then replace all `zap.S()` calls:

```go
// Import changes: remove "go.uber.org/zap", add:
// logger "github.com/theurichde/go-aws-sso/pkg/logger"

// Line 108: zap.S().Fatal(lockedAuthFlowMsg) → logger.L.Fatal(lockedAuthFlowMsg)
// Line 123: zap.S().Fatal(lockedAuthFlowMsg) → logger.L.Fatal(lockedAuthFlowMsg)
// Line 188: zap.S().Fatalf(...) → logger.L.Fatalf(...)
// Line 242: zap.S().Fatal(err) → logger.L.Fatal(err)
// Line 115: zap.S().Debugf(...) → logger.L.Debugf(...)
// Line 127: zap.S().Info(...) → logger.L.Info(...)
// Line 172: zap.S().Warnf(...) → logger.L.Warnf(...)
// Line 185: zap.S().Debugf(...) → logger.L.Debugf(...)
// Line 207: zap.S().Error(err) → logger.L.Error(err)
// Line 238: zap.S().Infof(...) → logger.L.Infof(...)
// Line 265: zap.S().Error(...) → logger.L.Error(...)
// Line 270: zap.S().Error(...) → logger.L.Error(...)
// Line 279: zap.S().Debug(...) → logger.L.Debug(...)
// Line 282: zap.S().Error(...) → logger.L.Error(...)
// Line 287: zap.S().Error(...) → logger.L.Error(...)
```

Replace `check` calls too:
  - Line 155: `check(err)` becomes inline `if err != nil { logger.L.Fatalf("Something went wrong: %q", err) }`
  Wait — `check` is defined in `file_system.go`. Let me handle this separately.

Actually, the `pkg/sso/aws.go` uses `check()` which is defined in `pkg/sso/file_system.go`. So:
- `registerClient` line 155: `check(err)` stays if we keep the file_system.go check (updated to use logger.L)
- `startDeviceAuthorization` line 171: `check(err)` same

Let me just do find-and-replace of `zap.S()` → `logger.L` and `"go.uber.org/zap"` → `logger` import. The check() functions will be updated separately in file_system.go.

- [ ] **Step 2: Replace in `pkg/sso/file_system.go`**

Replace import `"go.uber.org/zap"` with `logger "github.com/theurichde/go-aws-sso/pkg/logger"`.

```go
// Line 79:  zap.S().Debugf(...) → logger.L.Debugf(...)
// Line 84:  zap.S().Debugf(...) → logger.L.Debugf(...)
// Line 88:  zap.S().Debugf(...) → logger.L.Debugf(...)
// Line 101: zap.S().Panicf(...) → logger.L.Fatalf(...)  (Panicf→Fatalf: same unpredictable error, better to log and exit cleanly)
// Line 130: zap.S().Fatalf(...) → logger.L.Fatalf(...)
```

Also update the `check()` helper at line 128:
```go
func check(err error) {
	if err != nil {
		logger.L.Fatalf("Something went wrong: %q", err)
	}
}
```

- [ ] **Step 3: Replace in `pkg/sso/config.go`** (no zap import, no changes needed — already clean)

- [ ] **Step 4: Run tests**

```bash
go build ./...
go test ./pkg/sso/ -v
```

Expected: compile succeeds; tests pass. (Will fail if logger.L isn't initialized in tests — we'll fix this in Task 6.)

- [ ] **Step 5: Commit**

```bash
git add pkg/sso/aws.go pkg/sso/file_system.go
git commit -m "refactor(sso): replace zap.S() with logger.L"
```

---

### Task 3: Replace zap.S() in `internal/` with `logger.L`

**Files:**
- Modify: `internal/common.go`
- Modify: `internal/aws.go`
- Modify: `internal/assume.go`
- Modify: `internal/refresh.go`
- Modify: `internal/config.go`
- Modify: `internal/prompt.go`

- [ ] **Step 1: Update `internal/common.go`**

Replace import `"go.uber.org/zap"` with `logger "github.com/theurichde/go-aws-sso/pkg/logger"`.

```go
func check(err error) {
	if err != nil {
		logger.L.Fatalf("Something went wrong: %q", err)
	}
}
```

- [ ] **Step 2: Update `internal/aws.go`**

Replace import `"go.uber.org/zap"` with `logger "github.com/theurichde/go-aws-sso/pkg/logger"`.

```go
// Line 38: zap.S().Infof(...) → logger.L.Infof(...)
```

- [ ] **Step 3: Update `internal/assume.go`**

Replace import `"go.uber.org/zap"` with `logger "github.com/theurichde/go-aws-sso/pkg/logger"`.

```go
// Line 32: zap.S().Infof(...) → logger.L.Infof(...)
// Line 33: zap.S().Infof(...) → logger.L.Infof(...)
// Line 34: zap.S().Infof(...) → logger.L.Infof(...)
```

- [ ] **Step 4: Update `internal/refresh.go`**

Replace import `"go.uber.org/zap"` with `logger "github.com/theurichde/go-aws-sso/pkg/logger"`.

```go
// Line 32: zap.S().Infof(...) → logger.L.Infof(...)
// Line 40: zap.S().Info(...)  → logger.L.Info(...)
// Line 58: zap.S().Infof(...) → logger.L.Infof(...)
// Line 68: zap.S().Infof(...) → logger.L.Infof(...)
// Line 69: zap.S().Infof(...) → logger.L.Infof(...)
// Line 70: zap.S().Infof(...) → logger.L.Infof(...)
```

- [ ] **Step 5: Update `internal/config.go`**

Replace import `"go.uber.org/zap"` with `logger "github.com/theurichde/go-aws-sso/pkg/logger"`.

```go
// Line 90: zap.S().Infof(...) → logger.L.Infof(...)
```

- [ ] **Step 6: Update `internal/prompt.go`** — already has no zap import (uses `check()` which is in `common.go`), no changes needed.

- [ ] **Step 7: Run tests**

```bash
go build ./...
go test ./internal/ -v
```

Expected: compile succeeds; tests pass.

- [ ] **Step 8: Commit**

```bash
git add internal/common.go internal/aws.go internal/assume.go internal/refresh.go internal/config.go
git commit -m "refactor(internal): replace zap.S() with logger.L"
```

---

### Task 4: Replace zap.S() in `cmd/` and initialize logger

**Files:**
- Modify: `cmd/go-aws-sso/main.go`

- [ ] **Step 1: Update imports in `cmd/go-aws-sso/main.go`**

Remove `"go.uber.org/zap"` in import. Since `initializeLogger` still needs `zap` and `zapcore` for setup, keep those imports but add the logger import. Actually, `initializeLogger` needs `zap`, `zapcore` for setting up the pipeline — but we still use the global zap logger internally. We need both the zap import AND the logger import.

Keep: `"go.uber.org/zap"` and `"go.uber.org/zap/zapcore"` (for `initializeLogger`)
Add: `logger "github.com/theurichde/go-aws-sso/pkg/logger"`

- [ ] **Step 2: Update `initializeLogger()`**

```go
func initializeLogger(context *cli.Context) {
	if context.Bool("quiet") {
		logger.SetLogger(&logger.NoopLogger{})
		return
	}
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")

	config.ConsoleSeparator = " "
	config.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logLevel := zapcore.InfoLevel

	stdOut := zapcore.Lock(os.Stdout)
	stdErr := zapcore.Lock(os.Stderr)

	var options []zap.Option
	if context.Bool("debug") {
		logLevel = zapcore.DebugLevel
		config.EncodeCaller = zapcore.ShortCallerEncoder
		config.CallerKey = "callerKey"
		options = append(options, zap.WithCaller(true))
		options = append(options, zap.AddStacktrace(zap.ErrorLevel))
	}

	infoLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= logLevel && lvl <= zapcore.ErrorLevel
	})

	errorFatalLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl == zapcore.ErrorLevel || lvl == zapcore.FatalLevel
	})

	encoder := zapcore.NewConsoleEncoder(config)
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, stdOut, infoLevel),
		zapcore.NewCore(encoder, stdErr, errorFatalLevel))
	zapLogger := zap.New(core, options...)
	zap.ReplaceGlobals(zapLogger)
	logger.SetLogger(&logger.ZapLogger{})

	logger.L.Debug("Debug logging enabled")
}
```

- [ ] **Step 3: Replace remaining zap.S() calls in `cmd/go-aws-sso/main.go`**

```go
// Line 181: zap.S().Fatal(err) → logger.L.Fatal(err)
// Line 247 (check helper): zap.S().Fatalf(...) → logger.L.Fatalf(...)
// Line 252: zap.S().Debug(...) → logger.L.Debug(...)
// Line 254: zap.S().Warn(...) → logger.L.Warn(...)
// Line 269: zap.S().Infof(...) → logger.L.Infof(...)
// Line 271: zap.S().Infof(...) → logger.L.Infof(...)
// Line 273: zap.S().Infof(...) → logger.L.Infof(...)
// Line 275: zap.S().Debugf(...) → logger.L.Debugf(...)
// Line 277: zap.S().Infof(...) → logger.L.Infof(...)
```

- [ ] **Step 4: Run tests**

```bash
go build ./...
go test ./cmd/go-aws-sso/ -v
```

Expected: compile succeeds; tests pass.

- [ ] **Step 5: Commit**

```bash
git add cmd/go-aws-sso/main.go
git commit -m "refactor(cmd): replace zap.S() with logger.L, wire logger initialization"
```

---

### Task 5: Update test files for test-safe logger

**Files:**
- Modify: `internal/config_test.go`
- Modify: `pkg/sso/file_system_test.go`
- Modify: `internal/assume_test.go`
- Modify: `pkg/sso/aws_test.go`
- Modify: `cmd/go-aws-sso/main_test.go`

- [ ] **Step 1: Add `init()` or test setup to `pkg/sso/aws_test.go`**

Add a `TestMain` or at the top of the file:

```go
import "github.com/theurichde/go-aws-sso/pkg/logger"

func TestMain(m *testing.M) {
	logger.SetLogger(&logger.TestLogger{})
	os.Exit(m.Run())
}
```

Wait — `pkg/sso/aws_test.go` doesn't import `os` yet. Add it.

Actually, the cleanest approach is to set the logger in `TestMain` for each test package. But since `pkg/sso/aws_test.go` and `pkg/sso/file_system_test.go` are in the same package, one `TestMain` covers both.

Let's use `init()` instead, since there might not be a `TestMain` pattern:

For `pkg/sso/`: Add `TestMain` into a new file or into `aws_test.go` (it's fine to be the "main" test file). Since both files share the same package, one TestMain works for all tests.

For `internal/`: Add `TestMain` into `config_test.go` or `assume_test.go`.

For `cmd/`: `main_test.go` already replaces the zap globals, but needs to use `logger.SetLogger`.

Let me check: does `internal/config_test.go` already have `func TestMain`? No.

Let me be pragmatic and simply set the logger at the top of each test function, or use `init()`.

Actually, the cleanest approach for Go: use `TestMain` in each test package.

- [ ] **Step 2: Add `TestMain` to `pkg/sso/aws_test.go`**

At bottom of file, add:

```go
import "os"

func TestMain(m *testing.M) {
	logger.SetLogger(&logger.TestLogger{})
	os.Exit(m.Run())
}
```

Also need to add import for logger package:
```go
import "github.com/theurichde/go-aws-sso/pkg/logger"
```

- [ ] **Step 3: Add `TestMain` to `internal/config_test.go`**

At bottom of file, add:

```go
import (
	"os"
	"github.com/theurichde/go-aws-sso/pkg/logger"
)

func TestMain(m *testing.M) {
	logger.SetLogger(&logger.TestLogger{})
	os.Exit(m.Run())
}
```

Wait, but `internal/config_test.go` already imports `"os"` via the existing imports. Let me adjust.

Actually, let me re-read `internal/config_test.go`. It imports `"os"` already on line 7. Good.

- [ ] **Step 4: Update `cmd/go-aws-sso/main_test.go`**

The `Test_initializeLogger` test replaces zap globals with `&zap.Logger{}`. We need to ensure the global logger.L is reset after each test. The easiest way: just call `logger.SetLogger(&logger.TestLogger{})` before or after each test run.

Actually, for `main_test.go`, the tests test the actual `initializeLogger` function which calls `logger.SetLogger`. So the TestLogger should be fine as default. But `Test_initializeLogger` replaces zap globals — which is fine because the ZapLogger wraps zap.S() which uses the global.

Add to `main_test.go`:
```go
import "github.com/theurichde/go-aws-sso/pkg/logger"

func TestMain(m *testing.M) {
	logger.SetLogger(&logger.TestLogger{})
	os.Exit(m.Run())
}
```

- [ ] **Step 5: Add `TestMain` to `internal/assume_test.go`**

This file is in the same package as `config_test.go` ("internal"), so only ONE `TestMain` is needed across the whole package. Since we already added it to `config_test.go`, it will cover all `internal/` tests.

Wait, actually, Go requires only one `TestMain` per package. If we add it to `config_test.go` and then try to add it to `assume_test.go`, it'll be a duplicate definition error. So we add it to exactly one file per test package.

Let me put `TestMain` in `internal/config_test.go` (already planned) and NOT in `internal/assume_test.go`.

- [ ] **Step 6: Run all tests**

```bash
go test ./... -v
```

Expected: all tests pass. No panics from Fatal. No process exits.

- [ ] **Step 7: Commit**

```bash
git add internal/config_test.go pkg/sso/aws_test.go cmd/go-aws-sso/main_test.go
git commit -m "test: add TestMain to initialize TestLogger in each test package"
```

---

### Task 6: Final verification

- [ ] **Step 1: Run full test suite**

```bash
go test ./... -v -count=1
```

Expected: all packages pass.

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Run lint**

```bash
# Check if project has a lint config
go vet ./...
```

Expected: no warnings.

- [ ] **Step 4: Verify no residual zap.S() calls outside logger package and main.go**

```bash
grep -r "zap\.S()" --include="*.go" --exclude-dir=logger | grep -v "main.go\|_test.go"
```

Expected: no output (all zap.S() calls have been replaced except in the `main.go` `initializeLogger` function and test files).

Wait, test files also used `zap.S()` via `check()` calls. But `check()` now uses `logger.L`, so those should be fine. However, the test files themselves might still import `"go.uber.org/zap"` if they use other zap features. Let me check: `main_test.go` imports `zap` and `zapcore` for `Test_initializeLogger`. That's fine — `initializeLogger` still uses zap internally.

- [ ] **Step 5: Commit if no issues**

```bash
# No commit needed if everything already committed above
```
