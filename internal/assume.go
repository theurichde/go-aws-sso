package internal

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/sso/ssoiface"
	"github.com/aws/aws-sdk-go/service/ssooidc/ssooidciface"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"os"
	"time"
)

// AssumeDirectly
// Directly assumes into a certain account and role, bypassing the prompt and interactive selection.
func AssumeDirectly(oidcClient ssooidciface.SSOOIDCAPI, ssoClient ssoiface.SSOAPI, context *cli.Context) {
	startUrl := context.String("start-url")
	accountId := context.String("account-id")
	roleName := context.String("role-name")
	clientInformation, _ := ProcessClientInformation(oidcClient, startUrl)
	rci := &sso.GetRoleCredentialsInput{AccountId: &accountId, RoleName: &roleName, AccessToken: &clientInformation.AccessToken}
	roleCredentials, err := ssoClient.GetRoleCredentials(rci)
	check(err)

	if context.Bool("persist") {
		template := ProcessPersistedCredentialsTemplate(roleCredentials, context.String("profile"))
		WriteAWSCredentialsFile(template)

		zap.S().Infof("Successful retrieved credentials for account: %s", accountId)
		zap.S().Infof("Assumed role: %s", roleName)
		zap.S().Infof("Credentials expire at: %s\n", time.Unix(*roleCredentials.RoleCredentials.Expiration/1000, 0))
	} else {
		template := ProcessCredentialProcessTemplate(accountId, roleName, context.String("profile"), context.String("region"))
		WriteAWSCredentialsFile(template)

		creds := CredentialProcessOutput{
			Version:         1,
			AccessKeyId:     *roleCredentials.RoleCredentials.AccessKeyId,
			Expiration:      time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			SecretAccessKey: *roleCredentials.RoleCredentials.SecretAccessKey,
			SessionToken:    *roleCredentials.RoleCredentials.SessionToken,
		}
		bytes, _ := json.Marshal(creds)
		os.Stdout.Write(bytes)
	}

}

type CredentialProcessOutput struct {
	Version         int    `json:"Version"`
	AccessKeyId     string `json:"AccessKeyId"`
	Expiration      string `json:"Expiration"`
	SecretAccessKey string `json:"SecretAccessKey"`
	SessionToken    string `json:"SessionToken"`
}
