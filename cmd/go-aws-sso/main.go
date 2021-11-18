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
	"time"
)

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
			HasBeenSet:  IsFileExisting(homeDir + "/.aws/go-aws-sso-config.yaml"),
		},
	}

	commands := []*cli.Command{
		{
			Name:  "config",
			Usage: "Handles configuration",
			Subcommands: []*cli.Command{
				{
					Name:        "generate",
					Usage:       "Generates a config file with default values",
					Description: "Generates a config file. All available properties are set with a default value.\nOverrides any existing config file!\nUse --path to specify an alternative config file.\n  Defaults to ${HOME}/.aws/go-aws-sso-config.yaml",
					Action:      GenerateConfigAction,
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:    "path",
							Aliases: []string{"p"},
							Value:   homeDir + "/.aws/go-aws-sso-config.yaml",
						},
					},
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
			start(oidcApi, ssoApi, context)
			return nil
		},
		Flags:    flags,
		Commands: commands,
		Before:   altsrc.InitInputSourceWithContext(flags, altsrc.NewYamlSourceFromFlagFunc("config")),
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func start(oidcClient ssooidciface.SSOOIDCAPI, ssoClient ssoiface.SSOAPI, context *cli.Context) {

	startUrl := context.String("start-url")
	if startUrl == "" {
		log.Fatal("SSO start URL is not set.\nPlease use --start-url or set it via config file (see go-aws-sso config --help)")
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
		clientInformation = HandleOutdatedAccessToken(clientInformation, oidcClient)
	}

	// Accounts & Roles
	accountInfo, _ := RetrieveAccountInfo(clientInformation, ssoClient)
	roleInfo := RetrieveRoleInfo(accountInfo, clientInformation, ssoClient)

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
