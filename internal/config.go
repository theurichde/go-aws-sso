package internal

import (
	"os"
	"path/filepath"

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

	configFile := ConfigFilePath(context.String("config-path"))
	err := writeConfig(configFile, appConfig)
	return err
}

func EditConfigAction(context *cli.Context) error {

	configFile := ConfigFilePath(context.String("config-path"))
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return GenerateConfigAction(context)
	} else if err != nil {
		return err
	}
	config := ReadConfig(configFile)

	prompter := Prompter{}
	config.StartUrl = promptStartUrl(prompter, config.StartUrl)
	config.Region = promptRegion(prompter)

	err := writeConfig(configFile, *config)
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

	// Use OS-specific separators for Windows support.
	base := filepath.Dir(filePath)
	err = os.MkdirAll(base, 0755)
	check(err)

	err = os.WriteFile(filePath, bytes, 0755)
	check(err)

	zap.S().Infof("Config file generated: %s", filePath)

	return err
}

func ConfigFilePath(configPath string) string {
	if configPath != "" {
		return configPath
	}

	configDir, err := os.UserConfigDir()
	check(err)
	// Use OS-specific separators for Windows support.
	return filepath.Join(configDir, "go-aws-sso", "config.yml")
}
