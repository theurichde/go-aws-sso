package internal

import (
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

func GenerateConfigAction(_ *cli.Context) error {
	configFile := ConfigFilePath()
	err := writeConfig(configFile)
	return err
}

func writeConfig(filePath string) error {
	bytes, err := yaml.Marshal(NewDefaultAppConfig())
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
