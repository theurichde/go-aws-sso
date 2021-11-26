package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/sso/ssoiface"
	"github.com/aws/aws-sdk-go/service/ssooidc/ssooidciface"
	. "github.com/theurichde/go-aws-sso/internal"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
	"log"
	"os"
	"strings"
	"time"
)

func main() {

	flags := []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "start-url",
			Aliases: []string{"u"},
			Usage:   "Set / override the SSO login start-url. (Example: https://my-login.awsapps.com/start#/)",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "region",
			Aliases: []string{"r"},
			Value:   "eu-central-1",
			Usage:   "Set / override the AWS region",
		}),
	}

	commands := []*cli.Command{
		{
			Name:  "config",
			Usage: "Handles configuration. Note: Config location defaults to ${HOME}/.aws/go-aws-sso-config.yaml",
			Subcommands: []*cli.Command{
				{
					Name:        "generate",
					Usage:       "Generate a config file",
					Description: "Generates a config file. All available properties are interactively prompted.\nOverrides the existing config file!",
					Action:      GenerateConfigAction,
				},
				{
					Name:        "edit",
					Usage:       "Edit the config file",
					Description: "Edit the config file. All available properties are interactively prompted.\nOverrides the existing config file!",
					Action:      EditConfigAction,
				},
			},
		},
	}

	app := &cli.App{
		Name:                 "go-aws-sso",
		Usage:                "Retrieve short-living credentials via AWS SSO & SSOOIDC",
		EnableBashCompletion: true,
		Action: func(context *cli.Context) error {
			if len(context.Args().Slice()) != 0 {
				fmt.Printf("Command not found: %s\n", context.Args().First())
				println("Try help or --help for usage")
				os.Exit(1)
			}
			oidcApi, ssoApi := InitClients(context.String("region"))
			start(oidcApi, ssoApi, context, Prompter{})
			return nil
		},
		Flags:    flags,
		Commands: commands,
		Before:   ReadConfigFile(flags),
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func ReadConfigFile(flags []cli.Flag) cli.BeforeFunc {
	return func(context *cli.Context) error {
		inputSource, err := altsrc.NewYamlSourceFromFile(ConfigFilePath())
		if err != nil {
			if strings.Contains(err.Error(), "because it does not exist.") {
				return nil
			}
		}
		if err != nil {
			return fmt.Errorf("Unable to create input source: inner error: \n'%v'", err.Error())
		}

		return altsrc.ApplyInputSourceValues(context, inputSource, flags)
	}
}

func start(oidcClient ssooidciface.SSOOIDCAPI, ssoClient ssoiface.SSOAPI, context *cli.Context, promptSelector Prompt) {

	startUrl := context.String("start-url")
	if startUrl == "" {
		log.Println("No Start URL given. Please set it now.")
		err := GenerateConfigAction(context)
		check(err)
	}

	clientInformation, err := ReadClientInformation(ClientInfoFileDestination())
	if err != nil {
		var clientInfoPointer *ClientInformation
		clientInfoPointer = RegisterClient(oidcClient, startUrl)
		clientInfoPointer = RetrieveToken(oidcClient, Time{}, clientInfoPointer)
		WriteClientInfoToFile(clientInfoPointer, ClientInfoFileDestination())
		clientInformation = *clientInfoPointer
	} else if clientInformation.IsExpired() {
		log.Println("AccessToken expired. Start retrieving a new AccessToken.")
		clientInformation = HandleOutdatedAccessToken(clientInformation, oidcClient, startUrl)
	}

	accountInfo := RetrieveAccountInfo(clientInformation, ssoClient, promptSelector)
	roleInfo := RetrieveRoleInfo(accountInfo, clientInformation, ssoClient, promptSelector)

	rci := &sso.GetRoleCredentialsInput{AccountId: accountInfo.AccountId, RoleName: roleInfo.RoleName, AccessToken: &clientInformation.AccessToken}
	roleCredentials, err := ssoClient.GetRoleCredentials(rci)
	check(err)

	template := ProcessCredentialsTemplate(roleCredentials)
	WriteAWSCredentialsFile(template)

	log.Printf("Credentials expire at: %s\n", time.Unix(*roleCredentials.RoleCredentials.Expiration/1000, 0))

}

func check(err error) {
	if err != nil {
		log.Fatalf("Something went wrong: %q", err)
	}
}
