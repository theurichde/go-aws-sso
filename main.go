package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/sso/ssoiface"
	"github.com/aws/aws-sdk-go/service/ssooidc"
	"github.com/aws/aws-sdk-go/service/ssooidc/ssooidciface"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
	"github.com/valyala/fasttemplate"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const grantType = "urn:ietf:params:oauth:grant-type:device_code"
const clientType = "public"
const clientName = "go-aws-sso"

var cliContext *cli.Context

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

func main() {

	homeDir, _ := os.UserHomeDir()

	flags := []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "start-url",
			Aliases: []string{"u"},
			Usage:   "Set the SSO login start-url. (Example: https://my-login.awsapps.com/start#/)",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "region",
			Aliases: []string{"r"},
			Value:   "eu-central-1",
			Usage:   "Set the AWS region",
		}),
		&cli.StringFlag{
			Name:        "config",
			Aliases:     []string{"c"},
			Usage:       "Specify the config file to read from.",
			DefaultText: "~/.aws/go-aws-sso-config.yaml",
			Value:       homeDir + "/.aws/go-aws-sso-config.yaml",
			HasBeenSet:  isFileExisting(homeDir + "/.aws/go-aws-sso-config.yaml"),
		},
	}

	app := &cli.App{
		Name:  "go-aws-sso",
		Usage: "Retrieve short-living credentials via AWS SSO & SSOOIDC",
		Action: func(context *cli.Context) error {
			oidcApi, ssoApi := initClients(context.String("region"))
			start(oidcApi, ssoApi, context)
			return nil
		},
		Flags:  flags,
		Before: altsrc.InitInputSourceWithContext(flags, altsrc.NewYamlSourceFromFlagFunc("config")),
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func start(oidcClient ssooidciface.SSOOIDCAPI, ssoClient ssoiface.SSOAPI, context *cli.Context) {
	cliContext = context
	clientInformation, err := readClientInformation(clientInfoFileDestination())
	if err != nil {
		var clientInfoPointer *ClientInformation
		clientInfoPointer = registerClient(oidcClient)
		clientInfoPointer = retrieveToken(oidcClient, Time{}, clientInfoPointer)
		writeClientInfoToFile(clientInfoPointer, clientInfoFileDestination())
		clientInformation = *clientInfoPointer
	} else if clientInformation.isExpired() {
		log.Println("AccessToken expired. Start retrieving a new AccessToken.")
		clientInformation = handleOutdatedAccessToken(clientInformation, oidcClient)
	}

	// Accounts & Roles
	accountInfo, _ := retrieveAccountInfo(clientInformation, ssoClient)
	roleInfo := retrieveRoleInfo(accountInfo, clientInformation, ssoClient)

	rci := &sso.GetRoleCredentialsInput{AccountId: accountInfo.AccountId, RoleName: roleInfo.RoleName, AccessToken: &clientInformation.AccessToken}
	roleCredentials, err := ssoClient.GetRoleCredentials(rci)
	check(err)

	template := processCredentialsTemplate(roleCredentials)
	writeAWSCredentialsFile(template)

	log.Printf("Credentials expire at: %s\n", time.Unix(*roleCredentials.RoleCredentials.Expiration/1000, 0))

}

func processCredentialsTemplate(credentials *sso.GetRoleCredentialsOutput) string {
	template := `[default]
aws_access_key_id = {{access_key_id}}
aws_secret_access_key = {{secret_access_key}}
aws_session_token = {{session_token}}
output = json
region = eu-central-1`

	engine := fasttemplate.New(template, "{{", "}}")
	filledTemplate := engine.ExecuteString(map[string]interface{}{
		"access_key_id":     *credentials.RoleCredentials.AccessKeyId,
		"secret_access_key": *credentials.RoleCredentials.SecretAccessKey,
		"session_token":     *credentials.RoleCredentials.SessionToken,
	})
	return filledTemplate
}

func writeAWSCredentialsFile(template string) {

	homeDir, err := os.UserHomeDir()
	check(err)

	credentialsFile := homeDir + "/.aws/credentials"
	if !isFileExisting(credentialsFile) {
		err = os.MkdirAll(homeDir+"/.aws", 0777)
		check(err)
		_, err = os.OpenFile(credentialsFile, os.O_CREATE, 0644)
		check(err)
	}
	err = ioutil.WriteFile(credentialsFile, []byte(template), 0644)
	check(err)
}

func retrieveAccountInfo(clientInformation ClientInformation, ssoClient ssoiface.SSOAPI) (*sso.AccountInfo, error) {
	lai := sso.ListAccountsInput{AccessToken: &clientInformation.AccessToken}
	accounts, _ := ssoClient.ListAccounts(&lai)

	var accountsToSelect []string
	linePrefix := "#"

	for i, info := range accounts.AccountList {
		accountsToSelect = append(accountsToSelect, linePrefix+strconv.Itoa(i)+" "+*info.AccountName+" "+*info.AccountId)
	}

	prompt := promptui.Select{
		Label:             "Select your account - Hint: fuzzy search supported. To choose one account directly just enter #{Int}",
		Items:             accountsToSelect,
		Size:              20,
		Searcher:          fuzzySearchWithPrefixAnchor(accountsToSelect, linePrefix),
		StartInSearchMode: true,
	}

	indexChoice, _, err := prompt.Run()
	check(err)

	fmt.Println()

	accountInfo := accounts.AccountList[indexChoice]

	log.Printf("Selected account: %s - %s", *accountInfo.AccountName, *accountInfo.AccountId)
	fmt.Println()
	return accountInfo, err
}

func retrieveRoleInfo(accountInfo *sso.AccountInfo, clientInformation ClientInformation, ssoClient ssoiface.SSOAPI) *sso.RoleInfo {
	lari := &sso.ListAccountRolesInput{AccountId: accountInfo.AccountId, AccessToken: &clientInformation.AccessToken}
	roles, _ := ssoClient.ListAccountRoles(lari)

	var rolesToSelect []string
	linePrefix := "#"

	for i, info := range roles.RoleList {
		rolesToSelect = append(rolesToSelect, linePrefix+strconv.Itoa(i)+" "+*info.RoleName)
	}

	prompt := promptui.Select{
		Label:             "Select your role - Hint: fuzzy search supported. To choose one role directly just enter #{Int}",
		Items:             rolesToSelect,
		Size:              20,
		Searcher:          fuzzySearchWithPrefixAnchor(rolesToSelect, linePrefix),
		StartInSearchMode: true,
	}

	if len(roles.RoleList) == 1 {
		log.Printf("Only one role available. Selected role: %s\n", *roles.RoleList[0].RoleName)
		return roles.RoleList[0]
	}

	indexChoice, _, err := prompt.Run()
	check(err)

	roleInfo := roles.RoleList[indexChoice]
	return roleInfo
}

func fuzzySearchWithPrefixAnchor(itemsToSelect []string, linePrefix string) func(input string, index int) bool {
	return func(input string, index int) bool {
		role := itemsToSelect[index]

		if strings.HasPrefix(input, linePrefix) {
			if strings.HasPrefix(role, input) {
				return true
			} else {
				return false
			}
		} else {
			if fuzzy.MatchFold(input, role) {
				return true
			}
		}
		return false
	}
}

func handleOutdatedAccessToken(clientInformation ClientInformation, oidcClient ssooidciface.SSOOIDCAPI) ClientInformation {
	registerClientOutput := ssooidc.RegisterClientOutput{ClientId: &clientInformation.ClientId, ClientSecret: &clientInformation.ClientSecret}
	clientInformation.DeviceCode = *startDeviceAuthorization(oidcClient, &registerClientOutput).DeviceCode
	var clientInfoPointer *ClientInformation
	clientInfoPointer = retrieveToken(oidcClient, Time{}, &clientInformation)
	writeClientInfoToFile(clientInfoPointer, clientInfoFileDestination())
	return *clientInfoPointer
}

func generateCreateTokenInput(clientInformation *ClientInformation) ssooidc.CreateTokenInput {
	gtp := grantType
	return ssooidc.CreateTokenInput{ClientId: &clientInformation.ClientId, ClientSecret: &clientInformation.ClientSecret, DeviceCode: &clientInformation.DeviceCode, GrantType: &gtp}
}

func writeClientInfoToFile(information *ClientInformation, dest string) {
	file, err := json.MarshalIndent(information, "", " ")
	check(err)
	_ = ioutil.WriteFile(dest, file, 0644)
	//check(err)
}

func readClientInformation(file string) (ClientInformation, error) {
	if isFileExisting(file) {
		clientInformation := ClientInformation{}
		content, _ := ioutil.ReadFile(clientInfoFileDestination())
		err := json.Unmarshal(content, &clientInformation)
		check(err)
		return clientInformation, nil
	}
	return ClientInformation{}, errors.New("no ClientInformation exists")
}

func isFileExisting(file string) bool {
	if _, err := os.Stat(file); err == nil {
		return true
	} else if os.IsNotExist(err) {
		return false
	} else {
		message := fmt.Sprintf("Could not determine is file %q exists or not. Exiting.", file)
		panic(message)
	}
}

func registerClient(oidc ssooidciface.SSOOIDCAPI) *ClientInformation {
	cn := clientName
	ct := clientType

	rci := ssooidc.RegisterClientInput{ClientName: &cn, ClientType: &ct}
	rco, err := oidc.RegisterClient(&rci)
	check(err)

	sdao := startDeviceAuthorization(oidc, rco)

	return &ClientInformation{
		ClientId:                *rco.ClientId,
		ClientSecret:            *rco.ClientSecret,
		ClientSecretExpiresAt:   strconv.FormatInt(*rco.ClientSecretExpiresAt, 10),
		DeviceCode:              *sdao.DeviceCode,
		VerificationUriComplete: *sdao.VerificationUriComplete,
	}
}

func startDeviceAuthorization(oidc ssooidciface.SSOOIDCAPI, rco *ssooidc.RegisterClientOutput) ssooidc.StartDeviceAuthorizationOutput {
	startUrl := cliContext.String("start-url")
	sdao, err := oidc.StartDeviceAuthorization(&ssooidc.StartDeviceAuthorizationInput{ClientId: rco.ClientId, ClientSecret: rco.ClientSecret, StartUrl: &startUrl})
	check(err)
	log.Println("Please verify your client request: " + *sdao.VerificationUriComplete)
	return *sdao
}

func retrieveToken(client ssooidciface.SSOOIDCAPI, timer Timer, info *ClientInformation) *ClientInformation {
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

func clientInfoFileDestination() string {
	homeDir, _ := os.UserHomeDir()
	return homeDir + "/.aws/sso/cache/access-token.json"
}

func (ati ClientInformation) isExpired() bool {
	if ati.AccessTokenExpiresAt.Before(time.Now()) {
		return true
	}
	return false
}

func initClients(region string) (ssooidciface.SSOOIDCAPI, ssoiface.SSOAPI) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.AnonymousCredentials},
	))

	oidcClient := ssooidc.New(sess, aws.NewConfig().WithRegion(region))
	ssoClient := sso.New(sess, aws.NewConfig().WithRegion(region))

	return oidcClient, ssoClient
}

func check(err error) {
	if err != nil {
		log.Fatalf("Something went wrong: %q", err)
	}
}
