// Package logger provides a logger abstraction over zap.
// The package-level L variable is used throughout the application.
// Tests can replace it with a TestLogger to avoid os.Exit on Fatal.
package logger

import (
	"fmt"

	"go.uber.org/zap"
)

// Logger defines the logging interface used throughout the application.
type Logger interface {
	Debug(args ...interface{})
	Debugf(template string, args ...interface{})
	Info(args ...interface{})
	Infof(template string, args ...interface{})
	Warn(args ...interface{})
	Warnf(template string, args ...interface{})
	Error(args ...interface{})
	Errorf(template string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(template string, args ...interface{})
}

// L is the application-wide logger. Set via SetLogger at startup.
// Defaults to NoopLogger (silent) until initialized.
var L Logger = &NoopLogger{}

// SetLogger replaces the global application logger.
func SetLogger(l Logger) {
	L = l
}

// ZapLogger implements Logger by wrapping zap.S().
type ZapLogger struct{}

func (z *ZapLogger) Debug(args ...interface{})                   { zap.S().Debug(args...) }
func (z *ZapLogger) Debugf(template string, args ...interface{})  { zap.S().Debugf(template, args...) }
func (z *ZapLogger) Info(args ...interface{})                    { zap.S().Info(args...) }
func (z *ZapLogger) Infof(template string, args ...interface{})   { zap.S().Infof(template, args...) }
func (z *ZapLogger) Warn(args ...interface{})                    { zap.S().Warn(args...) }
func (z *ZapLogger) Warnf(template string, args ...interface{})   { zap.S().Warnf(template, args...) }
func (z *ZapLogger) Error(args ...interface{})                   { zap.S().Error(args...) }
func (z *ZapLogger) Errorf(template string, args ...interface{})  { zap.S().Errorf(template, args...) }
func (z *ZapLogger) Fatal(args ...interface{})                   { zap.S().Fatal(args...) }
func (z *ZapLogger) Fatalf(template string, args ...interface{})  { zap.S().Fatalf(template, args...) }

// NoopLogger implements Logger by discarding all log messages.
type NoopLogger struct{}

func (n *NoopLogger) Debug(args ...interface{})                   {}
func (n *NoopLogger) Debugf(template string, args ...interface{}) {}
func (n *NoopLogger) Info(args ...interface{})                    {}
func (n *NoopLogger) Infof(template string, args ...interface{})  {}
func (n *NoopLogger) Warn(args ...interface{})                    {}
func (n *NoopLogger) Warnf(template string, args ...interface{})  {}
func (n *NoopLogger) Error(args ...interface{})                   {}
func (n *NoopLogger) Errorf(template string, args ...interface{}) {}
func (n *NoopLogger) Fatal(args ...interface{})                   {}
func (n *NoopLogger) Fatalf(template string, args ...interface{}) {}

// TestLogger implements Logger and panics on Fatal calls instead of
// calling os.Exit. Other log levels output via fmt for test visibility.
type TestLogger struct{}

func (t *TestLogger) Debug(args ...interface{})                   {}
func (t *TestLogger) Debugf(template string, args ...interface{}) {}
func (t *TestLogger) Info(args ...interface{})                    {}
func (t *TestLogger) Infof(template string, args ...interface{})  {}
func (t *TestLogger) Warn(args ...interface{})                    {}
func (t *TestLogger) Warnf(template string, args ...interface{})  {}
func (t *TestLogger) Error(args ...interface{}) {
	fmt.Println("ERROR:", fmt.Sprint(args...))
}
func (t *TestLogger) Errorf(template string, args ...interface{}) {
	fmt.Printf("ERROR: "+template+"\n", args...)
}
func (t *TestLogger) Fatal(args ...interface{}) {
	panic("FATAL: " + fmt.Sprint(args...))
}
func (t *TestLogger) Fatalf(template string, args ...interface{}) {
	panic(fmt.Sprintf("FATAL: "+template, args...))
}
