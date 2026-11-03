package internal

import (
	"flag"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigFilePath(t *testing.T) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, customPath := range []string{"", filepath.Join(t.TempDir(), "portal.yaml"), "./portal-b.yml"} {
		t.Run(customPath, func(t *testing.T) {
			want := customPath
			if want == "" {
				want = filepath.Join(configDir, "go-aws-sso", "config.yml")
			}
			if got := ConfigFilePath(customPath); got != want {
				t.Errorf("ConfigFilePath() = %q, want %q", got, want)
			}
		})
	}
}

func TestGenerateConfigInNestedDirectory(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "SSO configs", "portal", "config.yaml")
	flags := flag.NewFlagSet("generate", flag.ContinueOnError)
	flags.String("config-path", configPath, "")
	flags.String("start-url", "https://portal.awsapps.com/start", "")
	flags.String("region", "eu-central-1", "")
	if err := GenerateConfigAction(cli.NewContext(nil, flags, nil)); err != nil {
		t.Fatal(err)
	}
	config := ReadConfig(configPath)
	if config.StartUrl != "https://portal.awsapps.com/start" || config.Region != "eu-central-1" {
		t.Fatalf("unexpected generated config: %+v", config)
	}
}

func TestWriteConfig(t *testing.T) {
	type args struct {
		context *cli.Context
	}

	tempFile := "/tmp/go-aws-sso/generated-config.yaml"
	defer func(file string) {
		dir := path.Dir(file)
		err := os.RemoveAll(dir)
		fail(err, t)
	}(tempFile)

	flagSet := flag.NewFlagSet("path", flag.ContinueOnError)
	flagSet.String("path", tempFile, "")
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Should create a default config file",
			args:    args{context: cli.NewContext(nil, flagSet, nil)},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			wantAppConfig := AppConfig{
				StartUrl: "https://my-login.awsapps.com/start#/",
				Region:   "eu-central-1",
			}

			got := writeConfig(tempFile, wantAppConfig)
			if got != nil {
				t.Errorf("Not expected: %q", got)
			}

			configFile, err := os.Open(tempFile)
			fail(err, t)

			bytes, err := os.ReadFile(configFile.Name())
			fail(err, t)

			gotAppConfig := AppConfig{}
			err = yaml.Unmarshal(bytes, &gotAppConfig)
			fail(err, t)

			if !reflect.DeepEqual(gotAppConfig, wantAppConfig) {
				t.Errorf("got: %q, want: %q", gotAppConfig, wantAppConfig)
			}
		})
	}
}

func fail(err error, t *testing.T) {
	if err != nil {
		t.Errorf("unexpected error: %q", err)
	}
}
