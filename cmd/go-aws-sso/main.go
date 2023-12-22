package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/sso"
	"github.com/aws/aws-sdk-go/service/sso/ssoiface"
	"github.com/aws/aws-sdk-go/service/ssooidc/ssooidciface"
	. "github.com/theurichde/go-aws-sso/internal"
	. "github.com/theurichde/go-aws-sso/pkg/sso"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = time.Now().String()
)

func main() {

	configFlags := []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "start-url",
			Aliases: []string{"u"},
			Usage:   "set / override the SSO login start-url. (Example: https://my-login.awsapps.com/start#/)",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "region",
			Aliases: []string{"r"},
			Usage:   "set / override the AWS region",
		}),
	}

	initialFlags := []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "profile",
			Aliases: []string{"p"},
			Value:   "default",
			Usage:   "the profile name you want to set in your ~/.aws/credentials file",
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:  "persist",
			Usage: "whether or not you want to write your short-living credentials to ~/.aws/credentials",
		}),
		&cli.BoolFlag{
			Name:     "force",
			Usage:    "removes the temporary access token and forces the retrieval of a new token",
			Value:    false,
			Hidden:   false,
			Required: false,
		},
		&cli.BoolFlag{
			Name:     "debug",
			Usage:    "enables debug logging",
			Value:    false,
			Hidden:   false,
			Required: false,
		},
	}

	initialFlags = append(configFlags, initialFlags...)

	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Printf("Version: %s\nCommit: %s\nBuild Time: %s\n", version, commit, date)
	}

	commands := []*cli.Command{
		{
			Name:  "config",
			Usage: "Handles configuration. Note: Config location defaults to $HOME/$CONFIG_DIR/go-aws-sso/config.yml",
			Subcommands: []*cli.Command{
				{
					Name:        "generate",
					Usage:       "Generate a config file",
					Description: "Generates a config file. All available properties are interactively prompted if not set with command options.\nOverrides the existing config file!",
					Action:      GenerateConfigAction,
					Flags:       configFlags,
				},
				{
					Name:        "edit",
					Usage:       "Edit the config file",
					Description: "Edit the config file. All available properties are interactively prompted.\nOverrides the existing config file!",
					Action:      EditConfigAction,
				},
			},
		},
		{
			Name:        "refresh",
			Usage:       "Refresh your previously used credentials.",
			Description: "Refreshes the short living credentials based on your last account and role.",
			Action: func(context *cli.Context) error {
				initializeLogger(context)
				checkMandatoryFlags(context)
				applyForceFlag(context)
				oidcApi, ssoApi := InitClients(context.String("region"))
				RefreshCredentials(oidcApi, ssoApi, context)
				return nil
			},
			Before: readConfigFile(initialFlags),
			Flags:  initialFlags,
		},
		{
			Name:        "assume",
			Usage:       "Assume directly into an account and SSO role",
			Description: "Assume directly into an account and SSO role",
			Action: func(context *cli.Context) error {
				initializeLogger(context)
				checkMandatoryFlags(context)
				applyForceFlag(context)
				oidcApi, ssoApi := InitClients(context.String("region"))
				AssumeDirectly(oidcApi, ssoApi, context)
				return nil
			},
			Before: readConfigFile(initialFlags),
			Flags: append(initialFlags, []cli.Flag{
				altsrc.NewStringFlag(&cli.StringFlag{
					Name:    "role-name",
					Aliases: []string{"n"},
					Usage:   "The role name you want to assume",
				}),
				altsrc.NewStringFlag(&cli.StringFlag{
					Name:    "account-id",
					Aliases: []string{"a"},
					Usage:   "The account id where your role lives in",
				}),
				&cli.BoolFlag{
					Name:     "quiet",
					Usage:    "disables logger output",
					Aliases:  []string{"q", "non-interactive"},
					Value:    false,
					Hidden:   false,
					Required: false,
				},
			}...),
		},
	}

	app := &cli.App{
		Name:                 "go-aws-sso",
		Usage:                "Retrieve short-living credentials via AWS SSO & SSOOIDC",
		EnableBashCompletion: true,
		Action: func(context *cli.Context) error {

			initializeLogger(context)

			if len(context.Args().Slice()) != 0 {
				fmt.Printf("Command not found: %s\n", context.Args().First())
				println("Try help or --help for usage")
				os.Exit(1)
			}

			checkMandatoryFlags(context)

			oidcApi, ssoApi := InitClients(context.String("region"))
			applyForceFlag(context)
			start(oidcApi, ssoApi, context, Prompter{})
			return nil
		},
		Flags:    initialFlags,
		Commands: commands,
		Before:   readConfigFile(initialFlags),
		Version:  version,
	}

	err := app.Run(os.Args)
	if err != nil {
		zap.S().Fatal(err)
	}
}

func readConfigFile(flags []cli.Flag) cli.BeforeFunc {
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
	clientInformation := ProcessClientInformation(oidcClient, startUrl)

	accountInfo, awsErr := RetrieveAccountInfo(clientInformation, ssoClient, promptSelector)
	if awsErr != nil && awsErr.StatusCode() == 401 { // unauthorized
		clientInformation, accountInfo = retryWithNewClientCreds(oidcClient, ssoClient, startUrl, promptSelector)
	}

	roleInfo := RetrieveRoleInfo(accountInfo, clientInformation, ssoClient, promptSelector)
	SaveUsageInformation(accountInfo, roleInfo)

	rci := &sso.GetRoleCredentialsInput{AccountId: accountInfo.AccountId, RoleName: roleInfo.RoleName, AccessToken: &clientInformation.AccessToken}
	roleCredentials, err := ssoClient.GetRoleCredentials(rci)
	check(err)

	if context.Bool("persist") {
		template := ProcessPersistedCredentialsTemplate(roleCredentials, context.String("region"))
		WriteAWSCredentialsFile(&template, context.String("profile"))
		zap.S().Infof("Credentials expire at: %s\n", time.Unix(*roleCredentials.RoleCredentials.Expiration/1000, 0))
	} else {
		template := ProcessCredentialProcessTemplate(*accountInfo.AccountId, *roleInfo.RoleName, context.String("region"))
		WriteAWSCredentialsFile(&template, context.String("profile"))
	}

}

func retryWithNewClientCreds(oidcClient ssooidciface.SSOOIDCAPI, ssoClient ssoiface.SSOAPI, startUrl string, promptSelector Prompt) (ClientInformation, *sso.AccountInfo) {
	err := os.Remove(ClientInfoFileDestination())
	check(err)
	clientInformation := ProcessClientInformation(oidcClient, startUrl)
	accountInfo, awsErr := RetrieveAccountInfo(clientInformation, ssoClient, promptSelector)
	check(awsErr)
	return clientInformation, accountInfo
}

func check(err error) {
	if err != nil {
		zap.S().Fatalf("Something went wrong: %q", err)
	}
}

func checkMandatoryFlags(context *cli.Context) {
	zap.S().Debug("Checking mandatory flags")
	if context.String("start-url") == "" || context.String("region") == "" {
		zap.S().Warn("No Start URL given. Please set it now.")
		err := GenerateConfigAction(context)
		check(err)
		appConfig := ReadConfig(ConfigFilePath())
		err = context.Set("start-url", appConfig.StartUrl)
		check(err)
		err = context.Set("region", appConfig.Region)
		check(err)
	}
}

func applyForceFlag(context *cli.Context) {
	if context.Bool("force") {
		err := os.Remove(ClientInfoFileDestination())
		if err != nil {
			zap.S().Infof("Nothing to do, no temporary access token found")
		}
		zap.S().Infof("Removed temporary acces token")
		err = os.Remove(os.TempDir() + "/go-aws-sso.lock")
		if err != nil {
			zap.S().Debugf("Nothing to do, no temporary lock file found")
		}
		zap.S().Infof("Removed temporary lock file")
	}
}

func initializeLogger(context *cli.Context) {
	if context.Bool("quiet") {
		zap.ReplaceGlobals(zap.NewNop())
		return
	}
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")

	config.ConsoleSeparator = " "
	config.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logLevel := zapcore.InfoLevel

	stdOut := zapcore.Lock(os.Stdout)
	stdErr := zapcore.Lock(os.Stderr)

	var options []zap.Option
	if context.Bool("debug") {
		logLevel = zapcore.DebugLevel
		config.EncodeCaller = zapcore.ShortCallerEncoder
		config.CallerKey = "callerKey"
		options = append(options, zap.WithCaller(true))
		options = append(options, zap.AddStacktrace(zap.ErrorLevel))
	}

	infoLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= logLevel && lvl <= zapcore.ErrorLevel
	})

	errorFatalLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl == zapcore.ErrorLevel || lvl == zapcore.FatalLevel
	})

	encoder := zapcore.NewConsoleEncoder(config)
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, stdOut, infoLevel),
		zapcore.NewCore(encoder, stdErr, errorFatalLevel))
	logger := zap.New(core, options...)
	zap.ReplaceGlobals(logger)

	zap.S().Debug("Debug logging enabled")
}
