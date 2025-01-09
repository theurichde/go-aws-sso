package sso

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/sso/ssoiface"
	"github.com/aws/aws-sdk-go/service/ssooidc"
	"github.com/aws/aws-sdk-go/service/ssooidc/ssooidciface"
	"go.uber.org/zap"
)

const grantType = "urn:ietf:params:oauth:grant-type:device_code"
const clientType = "public"
const clientName = "go-aws-sso"
const lockedAuthFlowMsg = "There is already an authorization flow running. If you think that is wrong, try using --force"

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

func InitClients(region string) (ssooidciface.SSOOIDCAPI, ssoiface.SSOAPI) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.AnonymousCredentials},
	))

	oidcClient := ssooidc.New(sess, aws.NewConfig().WithRegion(region))
	ssoClient := sso.New(sess, aws.NewConfig().WithRegion(region))

	return oidcClient, ssoClient
}

func ClientInfoFileDestination() string {
	homeDir, _ := os.UserHomeDir()
	return homeDir + "/.aws/sso/cache/access-token.json"
}

func (ati ClientInformation) isExpired() bool {
	if ati.AccessTokenExpiresAt.Before(time.Now()) {
		return true
	}
	return false
}

// ProcessClientInformation tries to read available ClientInformation.
// If no ClientInformation is available, it creates new ones and writes this information to disk
// If the start url is overridden and differs from the previous one, a new Client is registered for the given start url.
// When the ClientInformation.AccessToken is expired, it starts retrieving a new AccessToken
func ProcessClientInformation(oidcClient ssooidciface.SSOOIDCAPI, startUrl string) ClientInformation {
	if isAuthorizationFlowLocked() {
		zap.S().Fatal(lockedAuthFlowMsg)
	}

	clientInformation, err := ReadClientInformation(ClientInfoFileDestination())
	if err != nil || clientInformation.StartUrl != startUrl {
		lockAuthorizationFlow()
		defer unlockAuthorizationFlow()
		zap.S().Debugf("Encountered error while reading client information: %s", err)
		var clientInfoPointer *ClientInformation
		clientInfoPointer = registerClient(oidcClient, startUrl)
		clientInfoPointer = retrieveToken(oidcClient, Time{}, clientInfoPointer)
		WriteStructToFile(clientInfoPointer, ClientInfoFileDestination())
		clientInformation = *clientInfoPointer
	} else if clientInformation.isExpired() {
		if isAuthorizationFlowLocked() {
			zap.S().Fatal(lockedAuthFlowMsg)
		} else {
			lockAuthorizationFlow()
			defer unlockAuthorizationFlow()
			zap.S().Info("AccessToken expired. Start retrieving a new AccessToken")
			clientInformation = handleOutdatedAccessToken(clientInformation, oidcClient, startUrl)
		}
	}
	return clientInformation
}

func handleOutdatedAccessToken(clientInformation ClientInformation, oidcClient ssooidciface.SSOOIDCAPI, startUrl string) ClientInformation {
	registerClientOutput := ssooidc.RegisterClientOutput{ClientId: &clientInformation.ClientId, ClientSecret: &clientInformation.ClientSecret}
	clientInformation.DeviceCode = *startDeviceAuthorization(oidcClient, &registerClientOutput, startUrl).DeviceCode
	var clientInfoPointer *ClientInformation
	clientInfoPointer = retrieveToken(oidcClient, Time{}, &clientInformation)
	WriteStructToFile(clientInfoPointer, ClientInfoFileDestination())
	return *clientInfoPointer
}

func generateCreateTokenInput(clientInformation *ClientInformation) ssooidc.CreateTokenInput {
	return ssooidc.CreateTokenInput{
		ClientId:     &clientInformation.ClientId,
		ClientSecret: &clientInformation.ClientSecret,
		DeviceCode:   &clientInformation.DeviceCode,
		GrantType:    aws.String(grantType),
	}
}

func registerClient(oidc ssooidciface.SSOOIDCAPI, startUrl string) *ClientInformation {
	rci := ssooidc.RegisterClientInput{ClientName: aws.String(clientName), ClientType: aws.String(clientType)}
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

	// ignore when useing headless mode
	if strings.Contains(strings.Join(os.Args, ","), "--headless") {
		zap.S().Infof("using headless mode")
		return
	}

	if env, ok := os.LookupEnv("BROWSER"); ok {
		zap.S().Debugf("using BROWSER environment variable: %s", env)
		err = exec.Command(env, url).Start()
		if err != nil {
			zap.S().Fatalf("error while opening browser: %s", err)
		}
		return
	}

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
		err = fmt.Errorf("could not open %s - unsupported platform. Please open the URL manually or use the BROWSER environemnt variable to point to your browser", url)
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
	if err == nil {
		osInfo := strings.ToLower(string(bytes))
		return strings.Contains(osInfo, "wsl")
	}
	return false
}

func retrieveToken(client ssooidciface.SSOOIDCAPI, timer Timer, info *ClientInformation) *ClientInformation {
	input := generateCreateTokenInput(info)
	for {
		cto, err := client.CreateToken(&input)
		if err != nil {
			var awsErr awserr.Error
			if errors.As(err, &awsErr) {
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

type lockfile struct {
	LockTime time.Time `json:"lockTime"`
}

func unlockAuthorizationFlow() {
	_ = os.Remove(os.TempDir() + "/go-aws-sso.lock")
}

func lockAuthorizationFlow() {
	lf := lockfile{LockTime: time.Now()}
	lockBytes, err := json.Marshal(lf)
	if err != nil {
		zap.S().Error("Something went wrong while marshalling the temporary lock file", err)
	}

	err = os.WriteFile(os.TempDir()+"/go-aws-sso.lock", lockBytes, 0644)
	if err != nil {
		zap.S().Error("Something went wrong writing the temporary lock file", err)
	}
}

func isAuthorizationFlowLocked() bool {
	lockBytes, err := os.ReadFile(os.TempDir() + "/go-aws-sso.lock")
	var pathError *os.PathError
	if err != nil {
		if errors.As(err, &pathError) {
			zap.S().Debug("No lock file found")
			return false
		}
		zap.S().Error("Something went wrong while reading the temporary lock file", err)
	}
	lf := lockfile{}
	err = json.Unmarshal(lockBytes, &lf)
	if err != nil {
		zap.S().Error("Something went wrong while unmarshalling the temporary lock file", err)
		return false
	}

	return time.Now().Before(lf.LockTime.Add(time.Minute))
}
