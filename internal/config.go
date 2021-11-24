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

func NewDefaultAppConfig() *AppConfig {
	return &AppConfig{
		StartUrl: "https://my-login.awsapps.com/start#/",
		Region:   "eu-central-1",
	}
}

func promptStartUrl(prompt Prompt) string {
	return prompt.Prompt("SSO Start URL", "")
}

func promptRegion(prompt Prompt) string {
	p := prompt.Select("AWS Region", AwsRegions, func(input string, index int) bool {
		target := AwsRegions[index]
		return fuzzy.MatchFold(input, target)
	})
	_, s, _ := p.Run()
	return s
}

func GenerateConfigAction(_ *cli.Context) error {

	prompter := Prompter{}
	startUrl := promptStartUrl(prompter)
	region := promptRegion(prompter)
	appConfig := AppConfig{
		StartUrl: startUrl,
		Region:   region,
	}

	configFile := ConfigFilePath()
	err := writeConfig(configFile, appConfig)
	return err
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
