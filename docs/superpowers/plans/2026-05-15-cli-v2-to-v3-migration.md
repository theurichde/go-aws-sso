# CLI v2 to v3 Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate `github.com/urfave/cli/v2` to `github.com/urfave/cli/v3` and `github.com/urfave/cli/v2/altsrc` to `github.com/urfave/cli-altsrc/v3`, updating all handler signatures, flag definitions, test patterns, and removing v2 from go.mod.

**Architecture:** The root `cli.App` becomes a `cli.Command` (v3 removes App, everything is a Command). Handler signatures change from `func(*cli.Context) error` to `func(context.Context, *cli.Command) error`. The `*cli.Context` type no longer exists; all flag-reading methods (`String`, `Bool`, `Args`) and `Set` are now on `*cli.Command`. Tests no longer use `cli.NewContext` + `flag.NewFlagSet`; instead they create `*cli.Command` with inline flags and call `cmd.Set()` to simulate values.

**Tech Stack:** Go 1.22, urfave/cli v3.8.0, urfave/cli-altsrc v3.1.0

---

## Key Findings from v3 Source Code Analysis

### VersionPrinter
v3 still uses `cli.VersionPrinter` as a package-level var, but the signature changed:
```go
// v2
cli.VersionPrinter = func(c *cli.Context) { ... }
// v3
cli.VersionPrinter = func(cmd *cli.Command) { ... }
```

### cli.Command.Set() Exists
`*cli.Command` has a `Set(name, value string) error` method (`command.go:475`). It calls `flag.Set(name, value)`. This replaces `context.Set("name", val)`.

### Testing Pattern
Instead of `cli.NewContext(nil, flagSet, nil)`:
```go
cmd := &cli.Command{
    Flags: []cli.Flag{
        &cli.StringFlag{Name: "start-url", Value: "https://..."},
        &cli.BoolFlag{Name: "debug", Value: false},
    },
}
cmd.Set("start-url", "https://my-login.awsapps.com/start#/")
// then: cmd.String("start-url") returns the value
```

### altsrc v3 Changes
- Module: `github.com/urfave/cli-altsrc/v3`
- No more `altsrc.NewStringFlag()` / `altsrc.NewBoolFlag()` wrappers
- No more `altsrc.ApplyInputSourceValues()` 
- Flags are defined as plain `&cli.StringFlag{...}` / `&cli.BoolFlag{...}`
- Config file values are injected via `cmd.Set(name, value)` inside a Before function (checking `!cmd.IsSet(name)` first to preserve CLI-override semantics)

### Signature Changes Summary
| v2 | v3 |
|---|---|
| `func(c *cli.Context) error` | `func(ctx context.Context, cmd *cli.Command) error` |
| `cli.BeforeFunc` = `func(*cli.Context) error` | `cli.BeforeFunc` = `func(context.Context, *cli.Command) (context.Context, error)` |
| `context.String("name")` | `cmd.String("name")` |
| `context.Bool("name")` | `cmd.Bool("name")` |
| `context.Set("name", val)` | `cmd.Set("name", val)` |
| `context.IsSet("name")` | `cmd.IsSet("name")` |
| `context.Args().Slice()` | `cmd.Args().Slice()` |
| `context.Args().First()` | `cmd.Args().First()` |
| `app.Run(os.Args)` | `cmd.Run(context.Background(), os.Args)` |
| `EnableBashCompletion` | `EnableShellCompletion` |
| `Subcommands` | `Commands` |
| `EnvVars: []string{"X"}` | `Sources: cli.EnvVars("X")` |

---

### Task 1: Update go.mod — Remove v2, Add altsrc v3

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Edit go.mod to remove v2, add v3 altsrc dependency**

```bash
cd /home/theurich/projects/go-aws-sso
```

Edit `go.mod`: Remove line `github.com/urfave/cli/v2 v2.27.7`. Add `github.com/urfave/cli-altsrc/v3 v3.1.0` to the require block.

The resulting `go.mod` require block should contain:
```
	github.com/aws/aws-sdk-go v1.55.8
	github.com/chzyer/readline v1.5.1
	github.com/lithammer/fuzzysearch v1.1.8
	github.com/manifoldco/promptui v0.9.0
	github.com/urfave/cli-altsrc/v3 v3.1.0
	github.com/urfave/cli/v3 v3.8.0
	go.uber.org/zap v1.27.1
	gopkg.in/ini.v1 v1.67.0
	gopkg.in/yaml.v3 v3.0.1
```

- [ ] **Step 2: Run go mod tidy**

```bash
go mod tidy
```
Expected: should succeed with exit code 0. It may remove unused indirect dependencies (e.g., cpuguy83/go-md2man, russross/blackfriday, xrash/smetrics) that were v2-only.

- [ ] **Step 3: Run go build to verify dependencies resolve**

```bash
go build ./...
```
Expected: FAIL (compilation errors due to v2 symbols not found). This is expected — we haven't updated source files yet.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "build: replace cli/v2 with cli/v3 and add cli-altsrc/v3 dependency"
```

---

### Task 2: Migrate internal/handler files (assume.go, refresh.go, config.go)

**Files:**
- Modify: `internal/assume.go`
- Modify: `internal/refresh.go`
- Modify: `internal/config.go`

- [ ] **Step 1: Update imports in all three files**

In `internal/assume.go`, change:
```go
	"github.com/urfave/cli/v2"
```
to:
```go
	"github.com/urfave/cli/v3"
```

Do the same in `internal/refresh.go` and `internal/config.go`.

- [ ] **Step 2: Update handler signatures**

**internal/assume.go** — `AssumeDirectly` is NOT a handler registered on a Command directly; it's called from inline Action functions in main.go and from tests. It takes `*cli.Context` as parameter. Change its signature:

Change:
```go
func AssumeDirectly(oidcClient ssooidciface.SSOOIDCAPI, ssoClient ssoiface.SSOAPI, context *cli.Context) {
```
To:
```go
func AssumeDirectly(oidcClient ssooidciface.SSOOIDCAPI, ssoClient ssoiface.SSOAPI, cmd *cli.Command) {
```

Replace all `context.` calls inside `AssumeDirectly` with `cmd.`:
- `context.String("start-url")` → `cmd.String("start-url")`
- `context.String("account-id")` → `cmd.String("account-id")`
- `context.String("role-name")` → `cmd.String("role-name")`
- `context.Bool("persist")` → `cmd.Bool("persist")`
- `context.String("region")` → `cmd.String("region")`
- `context.String("profile")` → `cmd.String("profile")`

**internal/refresh.go** — `RefreshCredentials` also takes `*cli.Context` as parameter.

Change:
```go
func RefreshCredentials(oidcClient ssooidciface.SSOOIDCAPI, ssoClient ssoiface.SSOAPI, context *cli.Context) {
```
To:
```go
func RefreshCredentials(oidcClient ssooidciface.SSOOIDCAPI, ssoClient ssoiface.SSOAPI, cmd *cli.Command) {
```

Replace all `context.` calls inside:
- `context.String("start-url")` → `cmd.String("start-url")`
- `context.String("region")` → `cmd.String("region")`
- `context.String("profile")` → `cmd.String("profile")`

**internal/config.go** — `GenerateConfigAction` and `EditConfigAction` are registered as Action handlers. Their signatures change:

Change:
```go
func GenerateConfigAction(context *cli.Context) error {
```
To:
```go
func GenerateConfigAction(ctx context.Context, cmd *cli.Command) error {
```

Change:
```go
func EditConfigAction(_ *cli.Context) error {
```
To:
```go
func EditConfigAction(ctx context.Context, _ *cli.Command) error {
```

In `GenerateConfigAction`, replace:
- `context.String("start-url")` → `cmd.String("start-url")`
- `context.String("region")` → `cmd.String("region")`

Also add `"context"` to the import block in `internal/config.go`.

- [ ] **Step 3: Commit**

```bash
git add internal/assume.go internal/refresh.go internal/config.go
git commit -m "refactor: update internal handlers to cli/v3 signatures"
```

---

### Task 3: Migrate cmd/go-aws-sso/main.go

**Files:**
- Modify: `cmd/go-aws-sso/main.go`

This is the largest change. We need to update imports, all handler signatures, CLI definition, and helper functions.

- [ ] **Step 1: Update imports**

Replace:
```go
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
```
With:
```go
	"github.com/urfave/cli/v3"
	"github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli-altsrc/v3/yaml"
```
Add `"context"` to the import block.

- [ ] **Step 2: Replace altsrc-wrapped flag definitions with plain flags**

Change `configFlags` from:
```go
	configFlags := []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "start-url",
			Aliases: []string{"u"},
			Usage:   "set / override the SSO login start-url. (Example: https://my-login.awsapps.com/start#/)",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "region",
			Aliases: []string{"r"},
			Usage:   "set / override the AWS region",
		}),
	}
```
To:
```go
	configFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "start-url",
			Aliases: []string{"u"},
			Usage:   "set / override the SSO login start-url. (Example: https://my-login.awsapps.com/start#/)",
		},
		&cli.StringFlag{
			Name:    "region",
			Aliases: []string{"r"},
			Usage:   "set / override the AWS region",
		},
	}
```

Change `initialFlags` — remove `altsrc.NewStringFlag` / `altsrc.NewBoolFlag` wrappers:
```go
	initialFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "profile",
			Aliases: []string{"p"},
			Value:   "default",
			Usage:   "the profile name you want to set in your ~/.aws/credentials file",
		},
		&cli.BoolFlag{
			Name:  "persist",
			Usage: "whether or not you want to write your short-living credentials to ~/.aws/credentials",
		},
		&cli.BoolFlag{
			Name:     "force",
			Usage:    "removes the temporary access token and forces the retrieval of a new token",
			Value:    false,
			Hidden:   false,
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "debug",
			Usage:    "enables debug logging",
			Value:    false,
			Hidden:   false,
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "headless",
			Usage:    "show the verification URL without opening it in a browser",
			Value:    false,
			Hidden:   false,
			Required: false,
		},
	}
```

Change `assume` subcommand's extra flags — remove the `altsrc.NewStringFlag` wrapper:
```go
				&cli.StringFlag{
					Name:    "role-name",
					Aliases: []string{"n"},
					Usage:   "The role name you want to assume",
				},
				&cli.StringFlag{
					Name:    "account-id",
					Aliases: []string{"a"},
					Usage:   "The account id where your role lives in",
				},
```

- [ ] **Step 3: Update VersionPrinter**

Change:
```go
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("Version: %s\nCommit: %s\nBuild Time: %s\n", version, commit, date)
	}
```
To:
```go
	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Printf("Version: %s\nCommit: %s\nBuild Time: %s\n", version, commit, date)
	}
```

- [ ] **Step 4: Rewrite readConfigFile**

This function previously returned a `cli.BeforeFunc` that used `altsrc.NewYamlSourceFromFile` and `altsrc.ApplyInputSourceValues`. In v3, we read the YAML manually and use `cmd.Set()`.

Change:
```go
func readConfigFile(flags []cli.Flag) cli.BeforeFunc {
	return func(context *cli.Context) error {
		inputSource, err := altsrc.NewYamlSourceFromFile(ConfigFilePath())
		if err != nil {
			if strings.Contains(err.Error(), "because it does not exist.") {
				return nil
			}
		}
		if err != nil {
			return fmt.Errorf("Unable to create input source: inner error: \n'%v'", err.Error())
		}

		return altsrc.ApplyInputSourceValues(context, inputSource, flags)
	}
}
```
To:
```go
func readConfigFile(cmd *cli.Command) error {
	configPath := ConfigFilePath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil // config file is optional
	}

	config := AppConfig{}
	if err := yamlUnmarshal(data, &config); err != nil {
		return fmt.Errorf("unable to parse config file: %w", err)
	}

	// Set config values only if not already provided via CLI flags
	if config.StartUrl != "" && !cmd.IsSet("start-url") {
		if err := cmd.Set("start-url", config.StartUrl); err != nil {
			return err
		}
	}
	if config.Region != "" && !cmd.IsSet("region") {
		if err := cmd.Set("region", config.Region); err != nil {
			return err
		}
	}
	return nil
}
```

Note: We need to add `"gopkg.in/yaml.v3"` import (aliased if needed) to main.go for the unmarshal call. Actually, we can use `AppConfig` from `internal` package which is already dot-imported. Let's add `yamlUnmarshal` via the existing `gopkg.in/yaml.v3` import.

Wait — main.go dot-imports `internal` which has `ReadConfig` already. We can simplify:

```go
func readConfigFile(cmd *cli.Command) error {
	configPath := ConfigFilePath()
	if _, err := os.ReadFile(configPath); err != nil {
		return nil // config file is optional
	}

	config := ReadConfig(configPath) // from internal package

	// Set config values only if not already provided via CLI flags
	if config.StartUrl != "" && !cmd.IsSet("start-url") {
		if err := cmd.Set("start-url", config.StartUrl); err != nil {
			return err
		}
	}
	if config.Region != "" && !cmd.IsSet("region") {
		if err := cmd.Set("region", config.Region); err != nil {
			return err
		}
	}
	return nil
}
```

This uses the existing `ReadConfig` function from internal. No new imports needed.

Also update the function signature — `readConfigFile` no longer takes `flags` parameter since we don't need to enumerate them (we know the two keys we care about).

- [ ] **Step 5: Rewrite the CLI definition (cli.App → cli.Command)**

Replace the `app := &cli.App{...}` block: change `cli.App` → `cli.Command`, `EnableBashCompletion` → `EnableShellCompletion`, `Subcommands` → `Commands`, update all handler signatures.

The complete main() body from `commands` definition to `app.Run` changes as follows. Replace:

```go
	commands := []*cli.Command{
		{
			Name:  "config",
			Usage: "Handles configuration. Note: Config location defaults to $HOME/$CONFIG_DIR/go-aws-sso/config.yml",
			Subcommands: []*cli.Command{
				{
					Name:        "generate",
					Usage:       "Generate a config file",
					Description: "Generates a config file. All available properties are interactively prompted if not set with command options.\nOverrides the existing config file!",
					Action:      GenerateConfigAction,
					Flags:       configFlags,
				},
				{
					Name:        "edit",
					Usage:       "Edit the config file",
					Description: "Edit the config file. All available properties are interactively prompted.\nOverrides the existing config file!",
					Action:      EditConfigAction,
				},
			},
		},
		{
			Name:        "refresh",
			Usage:       "Refresh your previously used credentials.",
			Description: "Refreshes the short living credentials based on your last account and role.",
			Action: func(context *cli.Context) error {
				initializeLogger(context)
				checkMandatoryFlags(context)
				applyForceFlag(context)
				oidcApi, ssoApi := InitClients(context.String("region"))
				RefreshCredentials(oidcApi, ssoApi, context)
				return nil
			},
			Before: readConfigFile(initialFlags),
			Flags:  initialFlags,
		},
		{
			Name:        "assume",
			Usage:       "Assume directly into an account and SSO role",
			Description: "Assume directly into an account and SSO role",
			Action: func(context *cli.Context) error {
				initializeLogger(context)
				checkMandatoryFlags(context)
				applyForceFlag(context)
				oidcApi, ssoApi := InitClients(context.String("region"))
				AssumeDirectly(oidcApi, ssoApi, context)
				return nil
			},
			Before: readConfigFile(initialFlags),
			Flags: append(initialFlags, []cli.Flag{
				altsrc.NewStringFlag(&cli.StringFlag{
					Name:    "role-name",
					Aliases: []string{"n"},
					Usage:   "The role name you want to assume",
				}),
				altsrc.NewStringFlag(&cli.StringFlag{
					Name:    "account-id",
					Aliases: []string{"a"},
					Usage:   "The account id where your role lives in",
				}),
				&cli.BoolFlag{
					Name:     "quiet",
					Usage:    "disables logger output",
					Aliases:  []string{"q"},
					Value:    false,
					Hidden:   false,
					Required: false,
				},
			}...),
		},
	}

	app := &cli.App{
		Name:                 "go-aws-sso",
		Usage:                "Retrieve short-living credentials via AWS SSO & SSOOIDC",
		EnableBashCompletion: true,
		Action: func(context *cli.Context) error {

			initializeLogger(context)

			if len(context.Args().Slice()) != 0 {
				fmt.Printf("Command not found: %s\n", context.Args().First())
				println("Try help or --help for usage")
				os.Exit(1)
			}

			checkMandatoryFlags(context)

			oidcApi, ssoApi := InitClients(context.String("region"))
			applyForceFlag(context)
			start(oidcApi, ssoApi, context, Prompter{})
			return nil
		},
		Flags:    initialFlags,
		Commands: commands,
		Before:   readConfigFile(initialFlags),
		Version:  version,
	}

	err := app.Run(os.Args)
	if err != nil {
		zap.S().Fatal(err)
	}
```

With the full v3 version:

```go
	commands := []*cli.Command{
		{
			Name:  "config",
			Usage: "Handles configuration. Note: Config location defaults to $HOME/$CONFIG_DIR/go-aws-sso/config.yml",
			Commands: []*cli.Command{
				{
					Name:        "generate",
					Usage:       "Generate a config file",
					Description: "Generates a config file. All available properties are interactively prompted if not set with command options.\nOverrides the existing config file!",
					Action:      GenerateConfigAction,
					Flags:       configFlags,
				},
				{
					Name:        "edit",
					Usage:       "Edit the config file",
					Description: "Edit the config file. All available properties are interactively prompted.\nOverrides the existing config file!",
					Action:      EditConfigAction,
				},
			},
		},
		{
			Name:        "refresh",
			Usage:       "Refresh your previously used credentials.",
			Description: "Refreshes the short living credentials based on your last account and role.",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				initializeLogger(cmd)
				checkMandatoryFlags(cmd)
				applyForceFlag(cmd)
				oidcApi, ssoApi := InitClients(cmd.String("region"))
				RefreshCredentials(oidcApi, ssoApi, cmd)
				return nil
			},
			Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
				return ctx, readConfigFile(cmd)
			},
			Flags: initialFlags,
		},
		{
			Name:        "assume",
			Usage:       "Assume directly into an account and SSO role",
			Description: "Assume directly into an account and SSO role",
			Action: func(ctx context.Context, cmd *cli.Command) error {
				initializeLogger(cmd)
				checkMandatoryFlags(cmd)
				applyForceFlag(cmd)
				oidcApi, ssoApi := InitClients(cmd.String("region"))
				AssumeDirectly(oidcApi, ssoApi, cmd)
				return nil
			},
			Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
				return ctx, readConfigFile(cmd)
			},
			Flags: append(initialFlags, []cli.Flag{
				&cli.StringFlag{
					Name:    "role-name",
					Aliases: []string{"n"},
					Usage:   "The role name you want to assume",
				},
				&cli.StringFlag{
					Name:    "account-id",
					Aliases: []string{"a"},
					Usage:   "The account id where your role lives in",
				},
				&cli.BoolFlag{
					Name:     "quiet",
					Usage:    "disables logger output",
					Aliases:  []string{"q"},
					Value:    false,
					Hidden:   false,
					Required: false,
				},
			}...),
		},
	}

	cmd := &cli.Command{
		Name:                 "go-aws-sso",
		Usage:                "Retrieve short-living credentials via AWS SSO & SSOOIDC",
		EnableShellCompletion: true,
		Action: func(ctx context.Context, cmd *cli.Command) error {

			initializeLogger(cmd)

			if len(cmd.Args().Slice()) != 0 {
				fmt.Printf("Command not found: %s\n", cmd.Args().First())
				println("Try help or --help for usage")
				os.Exit(1)
			}

			checkMandatoryFlags(cmd)

			oidcApi, ssoApi := InitClients(cmd.String("region"))
			applyForceFlag(cmd)
			start(oidcApi, ssoApi, cmd, Prompter{})
			return nil
		},
		Flags:    initialFlags,
		Commands: commands,
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			return ctx, readConfigFile(cmd)
		},
		Version: version,
	}

	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		zap.S().Fatal(err)
	}
```

Important: add `"context"` to the import block.

- [ ] **Step 6: Update helper function signatures**

All functions taking `*cli.Context` must now take `*cli.Command`:

**`start`**:
```go
func start(oidcClient ssooidciface.SSOOIDCAPI, ssoClient ssoiface.SSOAPI, cmd *cli.Command, promptSelector Prompt) {
```
Replace `context.` → `cmd.` inside: `context.String("start-url")` → `cmd.String("start-url")`, `context.Bool("persist")` → `cmd.Bool("persist")`, `context.String("region")` → `cmd.String("region")`, `context.String("profile")` → `cmd.String("profile")`.

**`checkMandatoryFlags`**:
```go
func checkMandatoryFlags(cmd *cli.Command) {
```
Replace: `context.String("start-url")` → `cmd.String("start-url")`, `context.String("region")` → `cmd.String("region")`, `context.Set("start-url", ...)` → `cmd.Set("start-url", ...)`, `context.Set("region", ...)` → `cmd.Set("region", ...)`.

The `GenerateConfigAction(cmd)` call must be updated: change to `GenerateConfigAction(context.Background(), cmd)` since the signature now takes `context.Context` first. Actually, looking at it, `checkMandatoryFlags` is called from Action handlers that have access to `ctx context.Context`, but `checkMandatoryFlags` doesn't have a context parameter. Since it's called from Action functions that already have `ctx`, we should either:
1. Add `ctx context.Context` parameter to `checkMandatoryFlags`, or
2. Use `context.Background()` inside

Since `GenerateConfigAction` doesn't actually use `context.Context` for anything (it's just `_ *cli.Context` in v2), we can pass `context.Background()`:

```go
	err := GenerateConfigAction(context.Background(), cmd)
```

**`applyForceFlag`**:
```go
func applyForceFlag(cmd *cli.Command) {
```
Replace `context.Bool("force")` → `cmd.Bool("force")`.

**`initializeLogger`**:
```go
func initializeLogger(cmd *cli.Command) {
```
Replace `context.Bool("quiet")` → `cmd.Bool("quiet")`, `context.Bool("debug")` → `cmd.Bool("debug")`.

- [ ] **Step 7: Remove unused imports**

After all changes, verify that `"github.com/urfave/cli-altsrc/v3"` and `"github.com/urfave/cli-altsrc/v3/yaml"` are only imported if actually used. In the final code, `readConfigFile` uses `ReadConfig` from `internal` (dot-imported), so altsrc/yaml import may not be needed. Check:
- `altsrc` package: not used after removing `altsrc.NewStringFlag`, `altsrc.NewBoolFlag`, `altsrc.NewYamlSourceFromFile`, `altsrc.ApplyInputSourceValues`
- `altsrc/yaml` package: not used

So remove `"github.com/urfave/cli-altsrc/v3"` and `"github.com/urfave/cli-altsrc/v3/yaml"` imports. They can be removed from go.mod later when we add tests.

Wait — we still need `cli.ValueSourceChain` and `altsrc` for the env file source approach? No, the new `readConfigFile` reads YAML manually and calls `cmd.Set()`. No altsrc needed.

But let me reconsider: should we use the altsrc v3 pattern properly (with Sources on flags)? The cleaner v3 approach would be to attach a ValueSource to each flag's Sources chain in the Before. But that requires altsrc imports and is more complex. The `cmd.Set()` approach is simpler and preserves the v2 behavior (config file values only set if CLI didn't provide them). Let's stick with `cmd.Set()`. No altsrc imports needed in main.go.

- [ ] **Step 8: Commit**

```bash
git add cmd/go-aws-sso/main.go
git commit -m "refactor: migrate main.go to cli/v3 patterns"
```

---

### Task 4: Migrate test files — cmd/go-aws-sso/main_test.go

**Files:**
- Modify: `cmd/go-aws-sso/main_test.go`

This test file has `Test_start` and `Test_initializeLogger`. Both create `*cli.Context` via `cli.NewContext`.

- [ ] **Step 1: Update import**

Change:
```go
	"github.com/urfave/cli/v2"
```
To:
```go
	"github.com/urfave/cli/v3"
```
Remove `"flag"` import (no longer needed since we don't use `flag.NewFlagSet`).

- [ ] **Step 2: Update Test_start**

Replace the flag-based context creation:
```go
	flagSet := flag.NewFlagSet("start", 0)
	flagSet.String("start-url", "readConfigFile", "")
	flagSet.String("profile", "default", "")
	flagSet.String("region", "eu-central-1", "")
	flagSet.Bool("persist", true, "")

	newContext := cli.NewContext(nil, flagSet, nil)

	selector := mockPromptUISelector{}

	start(oidcClient, ssoClient, newContext, selector)
```

With:
```go
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "start-url", Value: ""},
			&cli.StringFlag{Name: "profile", Value: "default"},
			&cli.StringFlag{Name: "region", Value: "eu-central-1"},
			&cli.BoolFlag{Name: "persist", Value: false},
		},
	}
	_ = cmd.Set("start-url", "readConfigFile")
	_ = cmd.Set("profile", "default")
	_ = cmd.Set("region", "eu-central-1")
	_ = cmd.Set("persist", "true")

	selector := mockPromptUISelector{}

	start(oidcClient, ssoClient, cmd, selector)
```

- [ ] **Step 3: Update Test_initializeLogger**

Replace the flagset-based test setup. The test iterates over test cases with various flag combinations.

Replace:
```go
	for _, tt := range tests {
		zap.ReplaceGlobals(emptyLogger)
		t.Run(tt.name, func(t *testing.T) {
			flagSet := flag.NewFlagSet("test-set", flag.ContinueOnError)
			flagSet.Bool("debug", false, "")
			flagPtr := flagSet.Bool("quiet", false, "")
			flagSet.BoolVar(flagPtr, "q", false, "")
			flagSet.BoolVar(flagPtr, "non-interactive", false, "")

			err := flagSet.Parse(tt.flags)
			if err != nil {
				t.Fatal(err)
			}
			context := cli.NewContext(nil, flagSet, nil)

			initializeLogger(context)
```

With:
```go
	for _, tt := range tests {
		zap.ReplaceGlobals(emptyLogger)
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cli.Command{
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "debug", Value: false},
					&cli.BoolFlag{Name: "quiet", Aliases: []string{"q", "non-interactive"}, Value: false},
				},
			}
			// Parse test flags into the command
			for _, f := range tt.flags {
				switch f {
				case "--debug":
					_ = cmd.Set("debug", "true")
				case "--quiet":
					_ = cmd.Set("quiet", "true")
				case "-q":
					_ = cmd.Set("q", "true")
				case "--non-interactive":
					_ = cmd.Set("non-interactive", "true")
				}
			}

			initializeLogger(cmd)
```

Note: The `quiet` flag has aliases `q` and `non-interactive` in the original. In v3, the `Set` method needs a flag name. Since the aliases share the same BoolFlag, setting via any name sets the same underlying bool value. This should work.

- [ ] **Step 4: Commit**

```bash
git add cmd/go-aws-sso/main_test.go
git commit -m "test: migrate main_test.go to cli/v3 patterns"
```

---

### Task 5: Migrate test files — internal/assume_test.go

**Files:**
- Modify: `internal/assume_test.go`

- [ ] **Step 1: Update import**

Change:
```go
	"github.com/urfave/cli/v2"
```
To:
```go
	"github.com/urfave/cli/v3"
```
Remove `"flag"` import.

- [ ] **Step 2: Update TestAssumeDirectly**

Replace:
```go
	flagSet := flag.NewFlagSet("test-set", flag.ContinueOnError)
	flagSet.String("start-url", "foobar", "")
	flagSet.String("region", "eu-central-1", "")
	flagSet.String("account-id", "123456", "")
	flagSet.String("role-name", "super-admin", "")
	flagSet.String("profile", "default", "")
	flagSet.Bool("persist", true, "")
	ctx := cli.NewContext(nil, flagSet, nil)

	AssumeDirectly(oidcClient, ssoClient, ctx)
```

With:
```go
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "start-url", Value: ""},
			&cli.StringFlag{Name: "region", Value: ""},
			&cli.StringFlag{Name: "account-id", Value: ""},
			&cli.StringFlag{Name: "role-name", Value: ""},
			&cli.StringFlag{Name: "profile", Value: "default"},
			&cli.BoolFlag{Name: "persist", Value: false},
		},
	}
	_ = cmd.Set("start-url", "foobar")
	_ = cmd.Set("region", "eu-central-1")
	_ = cmd.Set("account-id", "123456")
	_ = cmd.Set("role-name", "super-admin")
	_ = cmd.Set("profile", "default")
	_ = cmd.Set("persist", "true")

	AssumeDirectly(oidcClient, ssoClient, cmd)
```

- [ ] **Step 3: Commit**

```bash
git add internal/assume_test.go
git commit -m "test: migrate assume_test.go to cli/v3 patterns"
```

---

### Task 6: Migrate test files — internal/config_test.go

**Files:**
- Modify: `internal/config_test.go`

- [ ] **Step 1: Update import**

Change:
```go
	"github.com/urfave/cli/v2"
```
To:
```go
	"github.com/urfave/cli/v3"
```
Remove `"flag"` import.

- [ ] **Step 2: Update TestWriteConfig**

The test currently creates a `*cli.Context` but only uses it as a test arg — the context is never actually used inside `writeConfig`. It looks like dead parameter cruft. The test creates a `flagSet` with a `"path"` flag but `writeConfig` doesn't even read it.

The test struct has `args{context: cli.NewContext(nil, flagSet, nil)}` but the test body doesn't use `tt.args.context` at all — it just calls `writeConfig(tempFile, wantAppConfig)` directly.

So we can simplify: remove the `args` struct with the context entirely, and remove the `cli` import. The test just needs:

```go
func TestWriteConfig(t *testing.T) {
	tempFile := "/tmp/go-aws-sso/generated-config.yaml"
	defer func(file string) {
		dir := path.Dir(file)
		err := os.RemoveAll(dir)
		fail(err, t)
	}(tempFile)

	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Should create a default config file",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			wantAppConfig := AppConfig{
				StartUrl: "https://my-login.awsapps.com/start#/",
				Region:   "eu-central-1",
			}

			got := writeConfig(tempFile, wantAppConfig)
			if got != nil {
				t.Errorf("Not expected: %q", got)
			}

			configFile, err := os.Open(tempFile)
			fail(err, t)

			bytes, err := os.ReadFile(configFile.Name())
			fail(err, t)

			gotAppConfig := AppConfig{}
			err = yaml.Unmarshal(bytes, &gotAppConfig)
			fail(err, t)

			if !reflect.DeepEqual(gotAppConfig, wantAppConfig) {
				t.Errorf("got: %q, want: %q", gotAppConfig, wantAppConfig)
			}
		})
	}
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/config_test.go
git commit -m "test: migrate config_test.go to cli/v3 patterns"
```

---

### Task 7: Final verification — build, test, lint

**Files:**
- (none modified, verification only)

- [ ] **Step 1: Run go build**

```bash
go build ./...
```
Expected: should succeed with exit code 0.

- [ ] **Step 2: Run go vet**

```bash
go vet ./...
```
Expected: should succeed with exit code 0.

- [ ] **Step 3: Run tests**

```bash
go test ./...
```
Expected: all tests should pass.

- [ ] **Step 4: Run go mod tidy (again)**

```bash
go mod tidy
```
This should clean up any unused indirect dependencies that were only needed by v2 (e.g., `cpuguy83/go-md2man`).

- [ ] **Step 5: Verify v2 is completely gone**

```bash
grep -r "urfave/cli/v2" --include="*.go" .
```
Expected: no matches.

- [ ] **Step 6: Commit**

```bash
git add .
git commit -m "chore: verify cli/v3 migration, run go mod tidy"
```

---

### Task 8: (Optional) Add altsrc v3 integration for config file sources

If we want to use the altsrc v3 pattern properly (attaching ValueSources to flags instead of manual `cmd.Set()`), we can add this as an enhancement after the base migration works. However, the current `cmd.Set()` approach in `readConfigFile` is simpler and preserves behavior correctly.

If desired later, the altsrc v3 approach would look like this in `readConfigFile`:
```go
func readConfigFile(cmd *cli.Command) error {
	configPath := ConfigFilePath()
	if _, err := os.ReadFile(configPath); err != nil {
		return nil
	}

	source := altsrc.NewValueSource(yaml.Unmarshal, "yaml", "", altsrc.StringSourcer(configPath))
	for _, flag := range cmd.Flags {
		if f, ok := flag.(*cli.StringFlag); ok && !f.IsSet() {
			if v, found := source.Lookup(); found {
				// This is more complex — need to map yaml keys to flag names
			}
		}
	}
	return nil
}
```
But this is not straightforward and the current approach works. Skip for now.

---

## Summary of Changes Per File

| File | Changes |
|---|---|
| `go.mod` | Remove v2, add cli-altsrc/v3 |
| `cmd/go-aws-sso/main.go` | Heavy: imports, app→command, handler sigs, readConfigFile rewrite, flag defs, all helper sigs |
| `cmd/go-aws-sso/main_test.go` | Imports, NewContext→Command+Set, parse flags manually |
| `internal/assume.go` | Import, handler param type, context.→cmd. |
| `internal/assume_test.go` | Import, NewContext→Command+Set |
| `internal/refresh.go` | Import, handler param type, context.→cmd. |
| `internal/config.go` | Import, handler sigs, context.→cmd., add context import |
| `internal/config_test.go` | Import, remove NewContext, simplify test |
