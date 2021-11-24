package internal

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/sso/ssoiface"
	"github.com/aws/aws-sdk-go/service/ssooidc"
	"github.com/aws/aws-sdk-go/service/ssooidc/ssooidciface"
	"log"
	"os"
	"strconv"
	"time"
)

const grantType = "urn:ietf:params:oauth:grant-type:device_code"
const clientType = "public"
const clientName = "go-aws-sso"

var AwsRegions = []string{
	"us-east-2",
	"us-east-1",
	"us-west-1",
	"us-west-2",
	"af-south-1",
	"ap-east-1",
	"ap-south-1",
	"ap-northeast-3",
	"ap-northeast-2",
	"ap-southeast-1",
	"ap-southeast-2",
	"ap-northeast-1",
	"ca-central-1",
	"eu-central-1",
	"eu-west-1",
	"eu-west-2",
	"eu-south-1",
	"eu-west-3",
	"eu-north-1",
	"me-south-1",
	"sa-east-1",
	"us-gov-east-1",
	"us-gov-west-1",
}

type ClientInformation struct {
	AccessTokenExpiresAt    time.Time
	AccessToken             string
	ClientId                string
	ClientSecret            string
	ClientSecretExpiresAt   string
	DeviceCode              string
	VerificationUriComplete string
}

type Timer interface {
	Now() time.Time
}

type Time struct {
}

func (i Time) Now() time.Time {
	return time.Now()
}

func ClientInfoFileDestination() string {
	homeDir, _ := os.UserHomeDir()
	return homeDir + "/.aws/sso/cache/access-token.json"
}

func (ati ClientInformation) IsExpired() bool {
	if ati.AccessTokenExpiresAt.Before(time.Now()) {
		return true
	}
	return false
}

func HandleOutdatedAccessToken(clientInformation ClientInformation, oidcClient ssooidciface.SSOOIDCAPI, startUrl string) ClientInformation {
	registerClientOutput := ssooidc.RegisterClientOutput{ClientId: &clientInformation.ClientId, ClientSecret: &clientInformation.ClientSecret}
	clientInformation.DeviceCode = *startDeviceAuthorization(oidcClient, &registerClientOutput, startUrl).DeviceCode
	var clientInfoPointer *ClientInformation
	clientInfoPointer = RetrieveToken(oidcClient, Time{}, &clientInformation)
	WriteClientInfoToFile(clientInfoPointer, ClientInfoFileDestination())
	return *clientInfoPointer
}

func generateCreateTokenInput(clientInformation *ClientInformation) ssooidc.CreateTokenInput {
	gtp := grantType
	return ssooidc.CreateTokenInput{ClientId: &clientInformation.ClientId, ClientSecret: &clientInformation.ClientSecret, DeviceCode: &clientInformation.DeviceCode, GrantType: &gtp}
}

func RegisterClient(oidc ssooidciface.SSOOIDCAPI, startUrl string) *ClientInformation {
	cn := clientName
	ct := clientType

	rci := ssooidc.RegisterClientInput{ClientName: &cn, ClientType: &ct}
	rco, err := oidc.RegisterClient(&rci)
	check(err)

	sdao := startDeviceAuthorization(oidc, rco, startUrl)

	return &ClientInformation{
		ClientId:                *rco.ClientId,
		ClientSecret:            *rco.ClientSecret,
		ClientSecretExpiresAt:   strconv.FormatInt(*rco.ClientSecretExpiresAt, 10),
		DeviceCode:              *sdao.DeviceCode,
		VerificationUriComplete: *sdao.VerificationUriComplete,
	}
}

func startDeviceAuthorization(oidc ssooidciface.SSOOIDCAPI, rco *ssooidc.RegisterClientOutput, startUrl string) ssooidc.StartDeviceAuthorizationOutput {
	sdao, err := oidc.StartDeviceAuthorization(&ssooidc.StartDeviceAuthorizationInput{ClientId: rco.ClientId, ClientSecret: rco.ClientSecret, StartUrl: &startUrl})
	check(err)
	log.Println("Please verify your client request: " + *sdao.VerificationUriComplete)
	return *sdao
}

func RetrieveToken(client ssooidciface.SSOOIDCAPI, timer Timer, info *ClientInformation) *ClientInformation {
	input := generateCreateTokenInput(info)
	for {
		cto, err := client.CreateToken(&input)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "AuthorizationPendingException" {
					log.Println("Still waiting for authorization...")
					time.Sleep(3 * time.Second)
					continue
				} else {
					log.Fatal(err)
				}
			}
		} else {
			info.AccessToken = *cto.AccessToken
			info.AccessTokenExpiresAt = timer.Now().Add(time.Hour*8 - time.Minute*5)
			return info
		}
	}
}

func InitClients(region string) (ssooidciface.SSOOIDCAPI, ssoiface.SSOAPI) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.AnonymousCredentials},
	))

	oidcClient := ssooidc.New(sess, aws.NewConfig().WithRegion(region))
	ssoClient := sso.New(sess, aws.NewConfig().WithRegion(region))

	return oidcClient, ssoClient
}

func RetrieveRoleInfo(accountInfo *sso.AccountInfo, clientInformation ClientInformation, ssoClient ssoiface.SSOAPI, selector Prompt) *sso.RoleInfo {
	lari := &sso.ListAccountRolesInput{AccountId: accountInfo.AccountId, AccessToken: &clientInformation.AccessToken}
	roles, _ := ssoClient.ListAccountRoles(lari)

	var rolesToSelect []string
	linePrefix := "#"

	for i, info := range roles.RoleList {
		rolesToSelect = append(rolesToSelect, linePrefix+strconv.Itoa(i)+" "+*info.RoleName)
	}

	label := "Select your role - Hint: fuzzy search supported. To choose one role directly just enter #{Int}"
	prompt := selector.Select(label, rolesToSelect, fuzzySearchWithPrefixAnchor(rolesToSelect, linePrefix))

	if len(roles.RoleList) == 1 {
		log.Printf("Only one role available. Selected role: %s\n", *roles.RoleList[0].RoleName)
		return roles.RoleList[0]
	}

	indexChoice, _, err := prompt.Run()
	check(err)

	roleInfo := roles.RoleList[indexChoice]
	return roleInfo
}

func RetrieveAccountInfo(clientInformation ClientInformation, ssoClient ssoiface.SSOAPI, selector Prompt) (*sso.AccountInfo, error) {
	lai := sso.ListAccountsInput{AccessToken: &clientInformation.AccessToken}
	accounts, _ := ssoClient.ListAccounts(&lai)

	var accountsToSelect []string
	linePrefix := "#"

	for i, info := range accounts.AccountList {
		accountsToSelect = append(accountsToSelect, linePrefix+strconv.Itoa(i)+" "+*info.AccountName+" "+*info.AccountId)
	}

	label := "Select your account - Hint: fuzzy search supported. To choose one account directly just enter #{Int}"
	prompt := selector.Select(label, accountsToSelect, fuzzySearchWithPrefixAnchor(accountsToSelect, linePrefix))

	indexChoice, _, err := prompt.Run()
	check(err)

	fmt.Println()

	accountInfo := accounts.AccountList[indexChoice]

	log.Printf("Selected account: %s - %s", *accountInfo.AccountName, *accountInfo.AccountId)
	fmt.Println()
	return accountInfo, err
}
