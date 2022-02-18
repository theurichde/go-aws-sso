package internal

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/sso/ssoiface"
	"github.com/aws/aws-sdk-go/service/ssooidc/ssooidciface"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"time"
)

type LastUsageInformation struct {
	AccountId   string `json:"account_id"`
	AccountName string `json:"account_name"`
	Role        string `json:"role"`
}

func RefreshCredentials(oidcClient ssooidciface.SSOOIDCAPI, ssoClient ssoiface.SSOAPI, context *cli.Context) {
	lui := readUsageInformation()

	startUrl := context.String("start-url")

	// TODO: refactor this part, lots of c&p from main.go start()

	clientInformation, err := ReadClientInformation(ClientInfoFileDestination())
	if err != nil {
		var clientInfoPointer *ClientInformation
		clientInfoPointer = RegisterClient(oidcClient, startUrl)
		clientInfoPointer = RetrieveToken(oidcClient, Time{}, clientInfoPointer)
		WriteStructToFile(clientInfoPointer, ClientInfoFileDestination())
		clientInformation = *clientInfoPointer
	} else if clientInformation.IsExpired() {
		log.Println("AccessToken expired. Start retrieving a new AccessToken.")
		clientInformation = HandleOutdatedAccessToken(clientInformation, oidcClient, startUrl)
	}

	rci := &sso.GetRoleCredentialsInput{AccountId: &lui.AccountId, RoleName: &lui.Role, AccessToken: &clientInformation.AccessToken}
	roleCredentials, err := ssoClient.GetRoleCredentials(rci)
	check(err)

	template := ProcessCredentialsTemplate(roleCredentials)
	WriteAWSCredentialsFile(template)

	// TODO: Error Handling when refresh file is not present

	log.Printf("Successful refreshed credentials for account: %s (%s)", lui.AccountName, lui.AccountId)
	log.Printf("Assumed role: %s", lui.Role)
	log.Printf("Credentials expire at: %s\n", time.Unix(*roleCredentials.RoleCredentials.Expiration/1000, 0))
}

func SaveUsageInformation(accountInfo *sso.AccountInfo, roleInfo *sso.RoleInfo) {
	homeDir, _ := os.UserHomeDir()
	target := homeDir + "/.aws/sso/cache/last-usage.json"
	usageInformation := LastUsageInformation{
		AccountId:   *accountInfo.AccountId,
		AccountName: *accountInfo.AccountName,
		Role:        *roleInfo.RoleName,
	}
	WriteStructToFile(usageInformation, target)
}

func readUsageInformation() *LastUsageInformation {
	bytes, err := os.ReadFile("/home/tim.heurich/.aws/sso/cache/last-usage.json")
	check(err)
	lui := new(LastUsageInformation)
	err = json.Unmarshal(bytes, lui)
	check(err)
	return lui
}
