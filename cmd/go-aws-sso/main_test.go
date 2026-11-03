package main

import (
	"flag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/sso/ssoiface"
	"github.com/aws/aws-sdk-go/service/ssooidc"
	"github.com/aws/aws-sdk-go/service/ssooidc/ssooidciface"
	. "github.com/theurichde/go-aws-sso/internal"
	. "github.com/theurichde/go-aws-sso/pkg/sso"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

func TestConfigPath(t *testing.T) {
	tests := []struct {
		name  string
		flags []string
		want  AppConfig
	}{
		{
			name: "settings from YAML",
			want: AppConfig{StartUrl: "https://portal.awsapps.com/start", Region: "eu-central-1"},
		},
		{
			name:  "CLI flags override YAML",
			flags: []string{"--start-url", "https://override.awsapps.com/start", "--region", "us-east-1"},
			want:  AppConfig{StartUrl: "https://override.awsapps.com/start", Region: "us-east-1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "portal.yaml")
			err := os.WriteFile(configPath, []byte("start-url: https://portal.awsapps.com/start\nregion: eu-central-1\n"), 0600)
			if err != nil {
				t.Fatal(err)
			}
			called := false
			action := func(context *cli.Context) error {
				called = true
				checkMandatoryFlags(context)
				if got := context.String("start-url"); got != tt.want.StartUrl {
					t.Errorf("start-url = %q, want %q", got, tt.want.StartUrl)
				}
				if got := context.String("region"); got != tt.want.Region {
					t.Errorf("region = %q, want %q", got, tt.want.Region)
				}
				return nil
			}
			flags := []cli.Flag{
				&cli.StringFlag{Name: "config-path"},
				altsrc.NewStringFlag(&cli.StringFlag{Name: "start-url"}),
				altsrc.NewStringFlag(&cli.StringFlag{Name: "region"}),
			}
			app := &cli.App{
				Flags:  flags,
				Before: readConfigFile(flags),
				Action: action,
			}
			args := []string{"go-aws-sso", "--config-path", configPath}
			args = append(args, tt.flags...)
			if err := app.Run(args); err != nil {
				t.Fatal(err)
			}
			if !called {
				t.Fatal("login action was not called")
			}
		})
	}
}

func TestMissingConfigPath(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    AppConfig
	}{
		{
			name: "root creates missing config",
			want: AppConfig{StartUrl: "https://portal.awsapps.com/start", Region: "eu-central-1"},
		},
		{
			name:    "assume creates missing config",
			command: "assume",
			want:    AppConfig{StartUrl: "https://portal.awsapps.com/start", Region: "eu-central-1"},
		},
		{
			name:    "refresh creates missing config",
			command: "refresh",
			want:    AppConfig{StartUrl: "https://portal.awsapps.com/start", Region: "eu-central-1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "nested", "portal.yaml")
			action := func(context *cli.Context) error {
				if _, err := os.Stat(configPath); !os.IsNotExist(err) {
					t.Fatalf("loading should not create a config, got %v", err)
				}
				if err := ensureConfigFile(context); err != nil {
					return err
				}
				config := ReadConfig(configPath)
				if config.StartUrl != context.String("start-url") || config.Region != context.String("region") {
					t.Fatalf("generated config differs from parsed settings: %+v", config)
				}
				return nil
			}
			configPathFlag := &cli.StringFlag{Name: "config-path"}
			flags := []cli.Flag{
				altsrc.NewStringFlag(&cli.StringFlag{Name: "start-url"}),
				altsrc.NewStringFlag(&cli.StringFlag{Name: "region"}),
			}
			app := &cli.App{
				Flags:  []cli.Flag{configPathFlag},
				Action: action,
			}
			args := []string{"go-aws-sso", "--config-path", configPath}
			if tt.command != "" {
				app.Before = readConfigFile(nil)
				app.Commands = []*cli.Command{{
					Name:   tt.command,
					Flags:  flags,
					Before: readConfigFile(flags),
					Action: action,
				}}
				args = append(args, tt.command)
			} else {
				app.Flags = append(app.Flags, flags...)
				app.Before = readConfigFile(flags)
			}
			settings := []string{"--start-url", tt.want.StartUrl, "--region", tt.want.Region}
			args = append(args, settings...)
			if err := app.Run(args); err != nil {
				t.Fatal(err)
			}
			config := ReadConfig(configPath)
			if *config != tt.want {
				t.Errorf("got: %+v, want: %+v", *config, tt.want)
			}
		})
	}
}

func TestMandatoryFlagsDoNotCreateConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "portal.yaml")
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	flags.String("config-path", configPath, "")
	flags.String("start-url", "https://portal.awsapps.com/start", "")
	flags.String("region", "eu-central-1", "")
	checkMandatoryFlags(cli.NewContext(nil, flags, nil))
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("mandatory flag check should not create a config, got %v", err)
	}
}

func TestInvalidConfigPath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "portal.yaml")
	if err := os.WriteFile(configPath, []byte("region: ["), 0600); err != nil {
		t.Fatal(err)
	}
	flags := []cli.Flag{
		&cli.StringFlag{Name: "config-path"},
		altsrc.NewStringFlag(&cli.StringFlag{Name: "start-url"}),
		altsrc.NewStringFlag(&cli.StringFlag{Name: "region"}),
	}
	app := &cli.App{
		Flags:  flags,
		Before: readConfigFile(flags),
		Action: func(*cli.Context) error {
			t.Fatal("invalid config must not reach login")
			return nil
		},
	}
	if err := app.Run([]string{"go-aws-sso", "--config-path", configPath}); err == nil {
		t.Fatal("expected invalid YAML error")
	}
}

type mockSSOOIDCClient struct {
	ssooidciface.SSOOIDCAPI
	CreateTokenOutput              ssooidc.CreateTokenOutput
	RegisterClientOutput           ssooidc.RegisterClientOutput
	StartDeviceAuthorizationOutput ssooidc.StartDeviceAuthorizationOutput
}

func (m mockSSOOIDCClient) CreateToken(_ *ssooidc.CreateTokenInput) (*ssooidc.CreateTokenOutput, error) {
	return &m.CreateTokenOutput, nil
}

func (m mockSSOOIDCClient) StartDeviceAuthorization(_ *ssooidc.StartDeviceAuthorizationInput) (*ssooidc.StartDeviceAuthorizationOutput, error) {
	return &m.StartDeviceAuthorizationOutput, nil
}

func (m mockSSOOIDCClient) RegisterClient(*ssooidc.RegisterClientInput) (*ssooidc.RegisterClientOutput, error) {
	return &m.RegisterClientOutput, nil
}

type mockSSOClient struct {
	ssoiface.SSOAPI
	GetRoleCredentialsOutput sso.GetRoleCredentialsOutput
	ListAccountRolesOutput   sso.ListAccountRolesOutput
	ListAccountsOutput       sso.ListAccountsOutput
}

func (m mockSSOClient) ListAccountRoles(_ *sso.ListAccountRolesInput) (*sso.ListAccountRolesOutput, error) {
	return &m.ListAccountRolesOutput, nil
}

func (m mockSSOClient) ListAccounts(_ *sso.ListAccountsInput) (*sso.ListAccountsOutput, error) {
	return &m.ListAccountsOutput, nil
}

func (m mockSSOClient) GetRoleCredentials(*sso.GetRoleCredentialsInput) (*sso.GetRoleCredentialsOutput, error) {
	return &m.GetRoleCredentialsOutput, nil
}

type mockTime struct {
	Timer
}

func (t mockTime) Now() time.Time {
	return time.Date(2021, 01, 01, 00, 00, 00, 00, &time.Location{})
}

func Test_start(t *testing.T) {
	os.Remove(os.TempDir() + "/go-aws-sso.lock")
	temp, err := os.CreateTemp("", "go-aws-sso_start")
	check(err)
	CredentialsFilePath = temp.Name()
	defer func(path string) {
		os.RemoveAll(path)
		os.Remove(os.TempDir() + "/go-aws-sso.lock")
	}(CredentialsFilePath)

	dummyInt := int64(132465)
	dummy := "dummy"
	accessToken := "AccessToken"
	accountId := "AccountId"
	accountName := "AccountName"
	roleName := "RoleName"
	accountId2 := "AccountId_2"
	accountName2 := "AccountName_2"
	roleName2 := "RoleName_2"

	ssoClient := mockSSOClient{
		SSOAPI: nil,
		GetRoleCredentialsOutput: sso.GetRoleCredentialsOutput{RoleCredentials: &sso.RoleCredentials{
			AccessKeyId:     &dummy,
			Expiration:      &dummyInt,
			SecretAccessKey: &dummy,
			SessionToken:    &dummy,
		}},
		ListAccountRolesOutput: sso.ListAccountRolesOutput{
			RoleList: []*sso.RoleInfo{
				{
					AccountId: &accountId,
					RoleName:  &roleName,
				},
				{
					AccountId: &accountId2,
					RoleName:  &roleName2,
				},
			},
		},
		ListAccountsOutput: sso.ListAccountsOutput{
			AccountList: []*sso.AccountInfo{
				{
					AccountId:   &accountId,
					AccountName: &accountName,
				},
				{
					AccountId:   &accountId2,
					AccountName: &accountName2,
				},
			},
		},
	}

	expires := int64(0)

	oidcClient := mockSSOOIDCClient{
		SSOOIDCAPI: nil,
		CreateTokenOutput: ssooidc.CreateTokenOutput{
			AccessToken: &accessToken,
		},
		RegisterClientOutput: ssooidc.RegisterClientOutput{
			AuthorizationEndpoint: &dummy,
			ClientId:              &dummy,
			ClientSecret:          &dummy,
			ClientSecretExpiresAt: &expires,
			TokenEndpoint:         &dummy,
		},
		StartDeviceAuthorizationOutput: ssooidc.StartDeviceAuthorizationOutput{
			DeviceCode:              &dummy,
			UserCode:                &dummy,
			VerificationUri:         &dummy,
			VerificationUriComplete: &dummy,
		},
	}

	_ = os.Setenv("HOME", "/tmp")

	flagSet := flag.NewFlagSet("start", 0)
	flagSet.String("start-url", "readConfigFile", "")
	flagSet.String("profile", "default", "")
	flagSet.String("region", "eu-central-1", "")
	flagSet.Bool("persist", true, "")

	newContext := cli.NewContext(nil, flagSet, nil)

	selector := mockPromptUISelector{}

	start(oidcClient, ssoClient, newContext, selector)

	content, _ := os.ReadFile(CredentialsFilePath)
	got := string(content)
	want := "[default]\naws_access_key_id     = dummy\naws_secret_access_key = dummy\naws_session_token     = dummy\nregion                = eu-central-1\n"

	if got != want {
		t.Errorf("Got: %v, but wanted: %v", got, want)
	}
}

type mockPromptUISelector struct {
}

func (receiver mockPromptUISelector) Select(_ string, _ []string, _ func(input string, index int) bool) (int, string) {
	return 0, ""
}

func (receiver mockPromptUISelector) Prompt(_ string, _ string) string {
	return ""
}

func Test_initializeLogger(t *testing.T) {
	type levelsEnabled struct {
		fatal bool
		error bool
		warn  bool
		info  bool
		debug bool
	}
	tests := []struct {
		name  string
		flags []string
		want  levelsEnabled
	}{
		{
			name:  "default",
			flags: []string{},
			want: levelsEnabled{
				fatal: true,
				error: true,
				warn:  true,
				info:  true,
				debug: false,
			},
		},
		{
			name:  "debug flag only",
			flags: []string{"--debug"},
			want: levelsEnabled{
				fatal: true,
				error: true,
				warn:  true,
				info:  true,
				debug: true,
			},
		},
		{
			name:  "quiet flag only",
			flags: []string{"--quiet"},
			want: levelsEnabled{
				fatal: false,
				error: false,
				warn:  false,
				info:  false,
				debug: false,
			},
		},
		{
			name:  "quiet flag alias only",
			flags: []string{"-q"},
			want: levelsEnabled{
				fatal: false,
				error: false,
				warn:  false,
				info:  false,
				debug: false,
			},
		},
		{
			name:  "quiet flag alternate alias only",
			flags: []string{"--non-interactive"},
			want: levelsEnabled{
				fatal: false,
				error: false,
				warn:  false,
				info:  false,
				debug: false,
			},
		},
		{
			name:  "quiet flag overrides debug flag",
			flags: []string{"--debug", "--quiet"},
			want: levelsEnabled{
				fatal: false,
				error: false,
				warn:  false,
				info:  false,
				debug: false,
			},
		},
	}
	// replace the zap logger with a temporary instance
	emptyLogger := &zap.Logger{}
	reset := zap.ReplaceGlobals(emptyLogger)
	defer reset()
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
			initializedLogger := zap.L()
			if initializedLogger == emptyLogger {
				t.Errorf("initializeLogger() did not initialize the logger")
			}
			// check if the logger is enabled for the desired levels
			gotLevels := levelsEnabled{
				fatal: initializedLogger.Core().Enabled(zapcore.FatalLevel),
				error: initializedLogger.Core().Enabled(zapcore.ErrorLevel),
				warn:  initializedLogger.Core().Enabled(zapcore.WarnLevel),
				info:  initializedLogger.Core().Enabled(zapcore.InfoLevel),
				debug: initializedLogger.Core().Enabled(zapcore.DebugLevel),
			}
			if tt.want != gotLevels {
				t.Errorf("Got: %v, but wanted: %v", gotLevels, tt.want)
			}
		})
	}
}
