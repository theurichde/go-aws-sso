package internal

import (
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
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

func GenerateConfigAction(context *cli.Context) error {
	bytes, err := yaml.Marshal(NewDefaultAppConfig())
	check(err)

	pathFlag := context.String("path")
	base := path.Dir(pathFlag)
	err = os.MkdirAll(base, 0755)
	check(err)

	file, err := os.OpenFile(pathFlag, os.O_CREATE|os.O_RDWR, 0755)
	check(err)

	_, err = file.Write(bytes)
	check(err)

	log.Printf("Config file generated: %s", file.Name())

	return err
}
