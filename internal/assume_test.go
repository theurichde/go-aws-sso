package internal

import (
	"flag"
	"os"
	"testing"

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

func (m mockSSOClient) GetRoleCredentials(*sso.GetRoleCredentialsInput) (*sso.GetRoleCredentialsOutput, error) {
	return &m.GetRoleCredentialsOutput, nil
}

func TestAssumeDirectly(t *testing.T) {

	temp, err := os.CreateTemp("", "go-aws-sso-assume-directly_")
	check(err)
	CredentialsFilePath = temp.Name()

	dummyInt := int64(132465)
	dummy := "dummy_assume_directly"
	accessToken := "AccessToken"

	ssoClient := mockSSOClient{
		SSOAPI: nil,
		GetRoleCredentialsOutput: sso.GetRoleCredentialsOutput{RoleCredentials: &sso.RoleCredentials{
			AccessKeyId:     &dummy,
			Expiration:      &dummyInt,
			SecretAccessKey: &dummy,
			SessionToken:    &dummy,
		}},
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

	flagSet := flag.NewFlagSet("test-set", flag.ContinueOnError)
	flagSet.String("start-url", "foobar", "")
	flagSet.String("region", "eu-central-1", "")
	flagSet.String("account-id", "123456", "")
	flagSet.String("role-name", "super-admin", "")
	flagSet.String("profile", "default", "")
	flagSet.Bool("persist", true, "")
	ctx := cli.NewContext(nil, flagSet, nil)

	AssumeDirectly(oidcClient, ssoClient, ctx)

	content, _ := os.ReadFile(CredentialsFilePath)
	defer os.RemoveAll(CredentialsFilePath)
	got := string(content)
	want := "[default]\naws_access_key_id     = dummy_assume_directly\naws_secret_access_key = dummy_assume_directly\naws_session_token     = dummy_assume_directly\nregion                = eu-central-1\n"

	if got != want {
		t.Errorf("Got: %v, but wanted: %v", got, want)
	}

}
