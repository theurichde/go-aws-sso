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
	"go.uber.org/zap"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
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
	StartUrl                string
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

// ProcessClientInformation tries to read available ClientInformation.
// If no ClientInformation is available it retrieves and creates new one and writes this information to disk
// If the start url is overridden via flag and differs from the previous one, a new Client is registered for the given start url.
// When the ClientInformation.AccessToken is expired, it starts retrieving a new AccessToken
func ProcessClientInformation(oidcClient ssooidciface.SSOOIDCAPI, startUrl string) (ClientInformation, error) {
	clientInformation, err := ReadClientInformation(ClientInfoFileDestination())
	if err != nil || clientInformation.StartUrl != startUrl {
		var clientInfoPointer *ClientInformation
		clientInfoPointer = RegisterClient(oidcClient, startUrl)
		clientInfoPointer = RetrieveToken(oidcClient, Time{}, clientInfoPointer)
		WriteStructToFile(clientInfoPointer, ClientInfoFileDestination())
		clientInformation = *clientInfoPointer
	} else if clientInformation.IsExpired() {
		zap.S().Infof("AccessToken expired. Start retrieving a new AccessToken.")
		clientInformation = HandleOutdatedAccessToken(clientInformation, oidcClient, startUrl)
	}
	return clientInformation, err
}

func HandleOutdatedAccessToken(clientInformation ClientInformation, oidcClient ssooidciface.SSOOIDCAPI, startUrl string) ClientInformation {
	registerClientOutput := ssooidc.RegisterClientOutput{ClientId: &clientInformation.ClientId, ClientSecret: &clientInformation.ClientSecret}
	clientInformation.DeviceCode = *startDeviceAuthorization(oidcClient, &registerClientOutput, startUrl).DeviceCode
	var clientInfoPointer *ClientInformation
	clientInfoPointer = RetrieveToken(oidcClient, Time{}, &clientInformation)
	WriteStructToFile(clientInfoPointer, ClientInfoFileDestination())
	return *clientInfoPointer
}

func generateCreateTokenInput(clientInformation *ClientInformation) ssooidc.CreateTokenInput {
	gtp := grantType
	return ssooidc.CreateTokenInput{
		ClientId:     &clientInformation.ClientId,
		ClientSecret: &clientInformation.ClientSecret,
		DeviceCode:   &clientInformation.DeviceCode,
		GrantType:    &gtp,
	}
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
		StartUrl:                startUrl,
	}
}

func startDeviceAuthorization(oidc ssooidciface.SSOOIDCAPI, rco *ssooidc.RegisterClientOutput, startUrl string) ssooidc.StartDeviceAuthorizationOutput {
	sdao, err := oidc.StartDeviceAuthorization(&ssooidc.StartDeviceAuthorizationInput{ClientId: rco.ClientId, ClientSecret: rco.ClientSecret, StartUrl: &startUrl})
	check(err)
	zap.S().Warnf("Please verify your client request: %s", *sdao.VerificationUriComplete)
	openUrlInBrowser(*sdao.VerificationUriComplete)
	return *sdao
}

func openUrlInBrowser(url string) {
	var err error
	osName := determineOsName()

	switch osName {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	case "wsl":
		err = exec.Command("wslview", url).Start()
	default:
		err = fmt.Errorf("could not open %s - unsupported platform. Please open the URL manually", url)
	}
	if err != nil {
		zap.S().Error(err)
	}
}

func determineOsName() string {
	if isWindowsSubsystemForLinuxOS() {
		return "wsl"
	}
	return runtime.GOOS
}

// isWindowsSubsystemForLinuxOS determines if the program is running on WSL
// Returns true if the OS is running in WSL, false if not.
// see https://github.com/microsoft/WSL/issues/423#issuecomment-844418910
func isWindowsSubsystemForLinuxOS() bool {
	bytes, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		osInfo := strings.ToLower(string(bytes))
		return strings.Contains("wsl", osInfo)
	}
	return false
}

func RetrieveToken(client ssooidciface.SSOOIDCAPI, timer Timer, info *ClientInformation) *ClientInformation {
	input := generateCreateTokenInput(info)
	for {
		cto, err := client.CreateToken(&input)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "AuthorizationPendingException" {
					zap.S().Infof("Still waiting for authorization...")
					time.Sleep(3 * time.Second)
					continue
				} else {
					zap.S().Fatal(err)
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

	if len(roles.RoleList) == 1 {
		zap.S().Infof("Only one role available. Selected role: %s\n", *roles.RoleList[0].RoleName)
		return roles.RoleList[0]
	}

	var rolesToSelect []string
	linePrefix := "#"

	for i, info := range roles.RoleList {
		rolesToSelect = append(rolesToSelect, linePrefix+strconv.Itoa(i)+" "+*info.RoleName)
	}

	label := "Select your role - Hint: fuzzy search supported. To choose one role directly just enter #{Int}"
	indexChoice, _ := selector.Select(label, rolesToSelect, fuzzySearchWithPrefixAnchor(rolesToSelect, linePrefix))
	roleInfo := roles.RoleList[indexChoice]
	return roleInfo
}

func RetrieveAccountInfo(clientInformation ClientInformation, ssoClient ssoiface.SSOAPI, selector Prompt) *sso.AccountInfo {
	var maxSize int64 = 1000 // default is 20, but sometimes you have more accounts available ;-)
	lai := sso.ListAccountsInput{AccessToken: &clientInformation.AccessToken, MaxResults: &maxSize}
	accounts, _ := ssoClient.ListAccounts(&lai)

	sortedAccounts := sortAccounts(accounts.AccountList)

	var accountsToSelect []string
	linePrefix := "#"

	for i, info := range sortedAccounts {
		accountsToSelect = append(accountsToSelect, linePrefix+strconv.Itoa(i)+" "+*info.AccountName+" "+*info.AccountId)
	}

	label := "Select your account - Hint: fuzzy search supported. To choose one account directly just enter #{Int}"
	indexChoice, _ := selector.Select(label, accountsToSelect, fuzzySearchWithPrefixAnchor(accountsToSelect, linePrefix))

	fmt.Println()

	accountInfo := sortedAccounts[indexChoice]

	zap.S().Infof("Selected account: %s - %s", *accountInfo.AccountName, *accountInfo.AccountId)
	fmt.Println()
	return &accountInfo
}

func sortAccounts(accountList []*sso.AccountInfo) []sso.AccountInfo {
	var sortedAccounts []sso.AccountInfo
	for _, info := range accountList {
		sortedAccounts = append(sortedAccounts, *info)
	}
	sort.Slice(sortedAccounts, func(i, j int) bool {
		return *sortedAccounts[i].AccountName < *sortedAccounts[j].AccountName
	})
	return sortedAccounts
}
