package main

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/ssooidc"
	"io/ioutil"
	"log"
	"os"
	"time"
)

const region = "eu-central-1"
const grantType = "urn:ietf:params:oauth:grant-type:device_code"

type AccessTokenInformation struct {
	ExpiresAt        time.Time
	TokenInformation ssooidc.CreateTokenOutput
}

func main() {

	// TODO: Reihenfolge muss noch geändert werden. Wenn AccessToken schon auf Platte existiert muss keine DevieAuth gestartet werden

	var clientName = "go-aws-sso-util"
	var clientType = "public"

	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.AnonymousCredentials},
	))

	oidc := ssooidc.New(sess, aws.NewConfig().WithRegion(region))
	rci := ssooidc.RegisterClientInput{ClientName: &clientName, ClientType: &clientType}
	rco, _ := oidc.RegisterClient(&rci)

	var startUrl = "https://idealo-login.awsapps.com/start#/"
	sdai := ssooidc.StartDeviceAuthorizationInput{ClientId: rco.ClientId, ClientSecret: rco.ClientSecret, StartUrl: &startUrl}
	sdao, _ := oidc.StartDeviceAuthorization(&sdai)

	println("Please verify your client request: " + *sdao.VerificationUriComplete)

	gtp := grantType
	cti := ssooidc.CreateTokenInput{ClientId: rco.ClientId, ClientSecret: rco.ClientSecret, DeviceCode: sdao.DeviceCode, GrantType: &gtp}

	var cto = retrieveToken(oidc, cti)

	ssoClient := sso.New(sess, aws.NewConfig().WithRegion(region))
	lai := sso.ListAccountsInput{AccessToken: cto.TokenInformation.AccessToken}
	accounts, _ := ssoClient.ListAccounts(&lai)
	for _, info := range accounts.AccountList {
		println(*info.AccountName)
	}

}

func retrieveToken(client *ssooidc.SSOOIDC, input ssooidc.CreateTokenInput) *AccessTokenInformation {
	// TODO: Check if token is expired
	if _, err := os.Stat(accessTokenFileDest()); err == nil {
		// read token from file
		tokenInformation := AccessTokenInformation{}
		content, _ := ioutil.ReadFile(accessTokenFileDest())
		_ = json.Unmarshal(content, &tokenInformation)
		return &tokenInformation
	} else if os.IsNotExist(err) {
		tokenInformation := tryToRetrieveToken(client, input)
		file, _ := json.MarshalIndent(tokenInformation, "", " ")
		_ = ioutil.WriteFile(accessTokenFileDest(), file, 0644)
		return tokenInformation
	} else {
		// TODO: file may or may not exist.
	}
	return tryToRetrieveToken(client, input)
}

func tryToRetrieveToken(client *ssooidc.SSOOIDC, input ssooidc.CreateTokenInput) *AccessTokenInformation {
	for {
		cto, err := client.CreateToken(&input)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "AuthorizationPendingException" {
					log.Println("Still waiting for authorization...")
					time.Sleep(3 * time.Second)
					continue
				}
			}
		} else {
			return &AccessTokenInformation{ExpiresAt: time.Now().Add(time.Hour*8 - time.Minute*15), TokenInformation: *cto}
		}
	}
}

func accessTokenFileDest() string {
	homeDir, _ := os.UserHomeDir()
	return homeDir + "/.aws/sso/cache/access-token.json"
}

func (ati AccessTokenInformation) isExpired() bool {
	if ati.ExpiresAt.Before(time.Now()) {
		return true
	}
	return false
}
