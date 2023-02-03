package internal

import (
	"go.uber.org/zap"
)

func check(err error) {
	if err != nil {
		zap.S().Fatal("Something went wrong: %q", err)
	}
}
