package internal

import (
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"path"
)

type AppConfig struct {
	StartUrl string `yaml:"start-url"`
	Region   string `yaml:"region"`
}

func promptStartUrl(prompt Prompt, dfault string) string {
	return prompt.Prompt("SSO Start URL", dfault)
}

func promptRegion(prompt Prompt) string {
	_, region := prompt.Select("AWS Region", AwsRegions, func(input string, index int) bool {
		target := AwsRegions[index]
		return fuzzy.MatchFold(input, target)
	})
	return region
}

func GenerateConfigAction(_ *cli.Context) error {

	prompter := Prompter{}
	startUrl := promptStartUrl(prompter, "")
	region := promptRegion(prompter)
	appConfig := AppConfig{
		StartUrl: startUrl,
		Region:   region,
	}

	configFile := ConfigFilePath()
	err := writeConfig(configFile, appConfig)
	return err
}

func EditConfigAction(_ *cli.Context) error {

	config := readConfig(ConfigFilePath())

	prompter := Prompter{}
	config.StartUrl = promptStartUrl(prompter, config.StartUrl)
	config.Region = promptRegion(prompter)

	err := writeConfig(ConfigFilePath(), *config)
	check(err)
	return err

}

func readConfig(filePath string) *AppConfig {

	bytes, err := ioutil.ReadFile(filePath)
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

	err = ioutil.WriteFile(filePath, bytes, 0755)
	check(err)

	log.Printf("Config file generated: %s", filePath)

	return err
}

func ConfigFilePath() string {
	homeDir, err := os.UserHomeDir()
	check(err)
	return homeDir + "/.aws/go-aws-sso-config.yaml"
}
