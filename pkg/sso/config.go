package sso

import "os"

// RuntimeConfig holds CLI-derived settings that affect behavior
// across the SSO flow. It is populated once at startup and treated
// as read-only for the remainder of the process.
type RuntimeConfig struct {
	Headless bool
}

// Config is the process-wide runtime configuration.
// It is set by the CLI entry points before any SSO logic executes.
var Config RuntimeConfig

// LoadRuntimeConfig initialises the global Config from the given CLI
// flag value, falling back to the GO_AWS_SSO_HEADLESS environment
// variable when the flag is false.
func LoadRuntimeConfig(headlessFlag bool) {
	Config = RuntimeConfig{
		Headless: headlessFlag || os.Getenv("GO_AWS_SSO_HEADLESS") != "",
	}
}
