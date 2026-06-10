package internal

import (
	"flag"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/theurichde/go-aws-sso/pkg/logger"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

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

func TestMain(m *testing.M) {
	logger.SetLogger(&logger.TestLogger{})
	os.Exit(m.Run())
}
