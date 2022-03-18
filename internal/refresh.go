package internal

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/sso/ssoiface"
	"github.com/aws/aws-sdk-go/service/ssooidc/ssooidciface"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"strings"
	"time"
)

type LastUsageInformation struct {
	AccountId   string `json:"account_id"`
	AccountName string `json:"account_name"`
	Role        string `json:"role"`
}

func RefreshCredentials(oidcClient ssooidciface.SSOOIDCAPI, ssoClient ssoiface.SSOAPI, context *cli.Context) {

	startUrl := context.String("start-url")
	clientInformation, _ := ProcessClientInformation(oidcClient, startUrl)

	var accountId *string
	var roleName *string

	lui, err := readUsageInformation()
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			log.Println("Nothing to refresh yet.")
			accountInfo := RetrieveAccountInfo(clientInformation, ssoClient, Prompter{})
			roleInfo := RetrieveRoleInfo(accountInfo, clientInformation, ssoClient, Prompter{})
			roleName = roleInfo.RoleName
			accountId = accountInfo.AccountId
			SaveUsageInformation(accountInfo, roleInfo)
		}
	} else {
		accountId = &lui.AccountId
		roleName = &lui.Role
	}

	rci := &sso.GetRoleCredentialsInput{AccountId: accountId, RoleName: roleName, AccessToken: &clientInformation.AccessToken}
	roleCredentials, err := ssoClient.GetRoleCredentials(rci)
	check(err)

	template := ProcessCredentialsTemplate(roleCredentials, context.String("profile"))
	WriteAWSCredentialsFile(template)

	log.Printf("Successful retrieved credentials for account: %s", *accountId)
	log.Printf("Assumed role: %s", *roleName)
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

func readUsageInformation() (*LastUsageInformation, error) {
	homeDir, _ := os.UserHomeDir()
	bytes, err := os.ReadFile(homeDir + "/.aws/sso/cache/last-usage.json")
	if err != nil {
		return nil, err
	}
	lui := new(LastUsageInformation)
	err = json.Unmarshal(bytes, lui)
	check(err)
	return lui, nil
}
