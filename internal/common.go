package internal

import (
	logger "github.com/theurichde/go-aws-sso/pkg/logger"
)

func check(err error) {
	if err != nil {
		logger.L.Fatalf("Something went wrong: %q", err)
	}
}
