package internal

import (
	"os"
	"path"

	"github.com/lithammer/fuzzysearch/fuzzy"
	. "github.com/theurichde/go-aws-sso/pkg/sso"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	StartUrl string `yaml:"start-url"`
	Region   string `yaml:"region"`
}

func promptStartUrl(prompt Prompt, dfault string) string {
	return prompt.Prompt("SSO Start URL", dfault)
}

func promptRegion(prompt Prompt) string {
	_, region := prompt.Select("Select your AWS Region. Hint: FuzzySearch supported", AwsRegions, func(input string, index int) bool {
		target := AwsRegions[index]
		return fuzzy.MatchFold(input, target)
	})
	return region
}

func GenerateConfigAction(context *cli.Context) error {

	prompter := Prompter{}
	startUrl := context.String("start-url")
	region := context.String("region")

	if startUrl == "" {
		startUrl = promptStartUrl(prompter, "")
	}

	if region == "" {
		region = promptRegion(prompter)
	}

	appConfig := AppConfig{
		StartUrl: startUrl,
		Region:   region,
	}

	configFile := ConfigFilePath()
	err := writeConfig(configFile, appConfig)
	return err
}

func EditConfigAction(_ *cli.Context) error {

	config := ReadConfig(ConfigFilePath())

	prompter := Prompter{}
	config.StartUrl = promptStartUrl(prompter, config.StartUrl)
	config.Region = promptRegion(prompter)

	err := writeConfig(ConfigFilePath(), *config)
	check(err)
	return err

}

func ReadConfig(filePath string) *AppConfig {

	bytes, err := os.ReadFile(filePath)
	check(err)
	appConfig := AppConfig{}
	err = yaml.Unmarshal(bytes, &appConfig)
	check(err)
	return &appConfig
}

func writeConfig(filePath string, ac AppConfig) error {
	bytes, err := yaml.Marshal(ac)
	check(err)

	base := path.Dir(filePath)
	err = os.MkdirAll(base, 0755)
	check(err)

	err = os.WriteFile(filePath, bytes, 0755)
	check(err)

	zap.S().Infof("Config file generated: %s", filePath)

	return err
}

func ConfigFilePath() string {
	configDir, err := os.UserConfigDir()
	check(err)
	return configDir + "/go-aws-sso/config.yml"
}
