package sso

import (
	"os"
	"testing"
)

func TestLoadRuntimeConfig(t *testing.T) {
	tests := []struct {
		name     string
		flag     bool
		envVal   string
		envSet   bool
		wantHead bool
	}{
		{
			name:     "flag true, env unset",
			flag:     true,
			envSet:   false,
			wantHead: true,
		},
		{
			name:     "flag false, env unset",
			flag:     false,
			envSet:   false,
			wantHead: false,
		},
		{
			name:     "flag false, env set to 1",
			flag:     false,
			envVal:   "1",
			envSet:   true,
			wantHead: true,
		},
		{
			name:     "flag false, env set to empty string",
			flag:     false,
			envVal:   "",
			envSet:   true,
			wantHead: false,
		},
		{
			name:     "flag true, env also set",
			flag:     true,
			envVal:   "1",
			envSet:   true,
			wantHead: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("GO_AWS_SSO_HEADLESS")
			Config = RuntimeConfig{}

			if tt.envSet {
				os.Setenv("GO_AWS_SSO_HEADLESS", tt.envVal)
				defer os.Unsetenv("GO_AWS_SSO_HEADLESS")
			}

			LoadRuntimeConfig(tt.flag)

			if Config.Headless != tt.wantHead {
				t.Errorf("Config.Headless = %v, want %v", Config.Headless, tt.wantHead)
			}
		})
	}
}
