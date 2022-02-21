package internal

import (
	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/sso/ssoiface"
	"github.com/aws/aws-sdk-go/service/ssooidc/ssooidciface"
	"github.com/urfave/cli/v2"
	"log"
	"time"
)

func AssumeDirectly(oidcClient ssooidciface.SSOOIDCAPI, ssoClient ssoiface.SSOAPI, context *cli.Context) {
	startUrl := context.String("start-url")
	accountId := context.String("account-id")
	roleName := context.String("role-name")
	clientInformation, err := ProcessClientInformation(oidcClient, startUrl)
	rci := &sso.GetRoleCredentialsInput{AccountId: &accountId, RoleName: &roleName, AccessToken: &clientInformation.AccessToken}
	roleCredentials, err := ssoClient.GetRoleCredentials(rci)
	check(err)
	template := ProcessCredentialsTemplate(roleCredentials)
	WriteAWSCredentialsFile(template)

	log.Printf("Successful retrieved credentials for account: %s", accountId)
	log.Printf("Assumed role: %s", roleName)
	log.Printf("Credentials expire at: %s\n", time.Unix(*roleCredentials.RoleCredentials.Expiration/1000, 0))
}
