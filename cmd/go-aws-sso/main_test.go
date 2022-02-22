package main

import (
	"flag"
	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/sso/ssoiface"
	"github.com/aws/aws-sdk-go/service/ssooidc"
	"github.com/aws/aws-sdk-go/service/ssooidc/ssooidciface"
	"github.com/theurichde/go-aws-sso/internal"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"
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
	internal.Timer
}

func (t mockTime) Now() time.Time {
	return time.Date(2021, 01, 01, 00, 00, 00, 00, &time.Location{})
}

func TestClientInformation_isExpired(t *testing.T) {
	type fields struct {
		AccessTokenExpiresAt time.Time
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "Should recognize a non-expired token",
			fields: fields{
				AccessTokenExpiresAt: time.Now().Add(time.Hour*8 - time.Minute*5),
			},
			want: false,
		},
		{
			name: "Should recognize an expired token",
			fields: fields{
				AccessTokenExpiresAt: time.Now().Add(-8 * time.Hour),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ati := internal.ClientInformation{
				AccessTokenExpiresAt: tt.fields.AccessTokenExpiresAt,
			}
			if got := ati.IsExpired(); got != tt.want {
				t.Errorf("isExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_retrieveToken(t *testing.T) {

	mockTime := mockTime{}
	at := "accessToken"
	want := internal.ClientInformation{AccessToken: at, AccessTokenExpiresAt: mockTime.Now().Add(time.Hour*8 - time.Minute*5)}

	t.Run("foobar", func(t *testing.T) {
		mockClient := mockSSOOIDCClient{CreateTokenOutput: ssooidc.CreateTokenOutput{
			AccessToken: &at,
		}}

		got := internal.RetrieveToken(mockClient, mockTime, &internal.ClientInformation{})

		if !reflect.DeepEqual(*got, want) {
			t.Errorf("retrieveToken() = got %v, want %v", *got, want)
		}

	})

}

func Test_processCredentialsTemplate(t *testing.T) {
	type args struct {
		accessKeyId     string
		expiration      string
		secretAccessKey string
		sessionToken    string
		credentials     *sso.GetRoleCredentialsOutput
	}

	accessKeyId := "access_key_id"
	secretAccessKey := "secret_access_key"
	sessionToken := "session_token"

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Process Credentials Template",
			args: args{
				credentials: &sso.GetRoleCredentialsOutput{RoleCredentials: &sso.RoleCredentials{
					AccessKeyId:     &accessKeyId,
					SecretAccessKey: &secretAccessKey,
					SessionToken:    &sessionToken,
				}},
			},
			want: "[default]\naws_access_key_id = access_key_id\naws_secret_access_key = secret_access_key\naws_session_token = session_token\noutput = json\nregion = eu-central-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := internal.ProcessCredentialsTemplate(tt.args.credentials, "default"); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("processCredentialsTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isFileExisting(t *testing.T) {

	tempFile, _ := os.CreateTemp("", "used-for-testing")
	t.Run("True if file exists", func(t *testing.T) {
		got := internal.IsFileOrFolderExisting(tempFile.Name())
		if got != true {
			t.Errorf("isFileExisting() = %v, want %v", got, true)
		}
	})

	t.Run("False if file does not exist", func(t *testing.T) {
		got := internal.IsFileOrFolderExisting("/tmp/not-existing-file.name")
		if got != false {
			t.Errorf("isFileExisting() = %v, want %v", got, true)
		}
	})
}

func Test_start(t *testing.T) {

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

	set := flag.NewFlagSet("start-url", 0)
	set.String("start-url", "ReadConfigFile", "")
	newContext := cli.NewContext(nil, set, nil)

	selector := mockPromptUISelector{}

	start(oidcClient, ssoClient, newContext, selector)

	homeDir, _ := os.UserHomeDir()
	content, _ := ioutil.ReadFile(homeDir + "/.aws/credentials")
	got := string(content)
	want := "[default]\naws_access_key_id = dummy\naws_secret_access_key = dummy\naws_session_token = dummy\noutput = json\nregion = eu-central-1"

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
