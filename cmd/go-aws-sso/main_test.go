package main

import (
	"flag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/sso/ssoiface"
	"github.com/aws/aws-sdk-go/service/ssooidc"
	"github.com/aws/aws-sdk-go/service/ssooidc/ssooidciface"
	. "github.com/theurichde/go-aws-sso/pkg/sso"
	"github.com/urfave/cli/v2"
)

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
