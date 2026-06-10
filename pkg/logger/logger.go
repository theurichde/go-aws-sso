// Package logger provides a logger abstraction over zap.
// The package-level L variable is used throughout the application.
// Tests can replace it with a TestLogger to avoid os.Exit on Fatal.
package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	Panic(args ...interface{})
	Panicf(template string, args ...interface{})
}

// L is the application-wide logger. Set via SetLogger at startup.
// Defaults to QuietLogger (silent) until initialized.
// Not safe for concurrent use; SetLogger should be called once before
// any goroutines that use L are started.
var L Logger = &QuietLogger{}

// SetLogger replaces the global application logger.
func SetLogger(l Logger) {
	L = l
}

// ZapLogger implements Logger by wrapping a *zap.SugaredLogger.
type ZapLogger struct {
	sugar *zap.SugaredLogger
}

// NewZapLogger creates a ZapLogger from a *zap.SugaredLogger.
func NewZapLogger(sugar *zap.SugaredLogger) *ZapLogger {
	return &ZapLogger{sugar: sugar}
}

// IsLevelEnabled reports whether the given level is enabled on the underlying logger.
func (z *ZapLogger) IsLevelEnabled(level zapcore.Level) bool {
	return z.sugar.Desugar().Core().Enabled(level)
}

func (z *ZapLogger) Debug(args ...interface{})                   { z.sugar.Debug(args...) }
func (z *ZapLogger) Debugf(template string, args ...interface{})  { z.sugar.Debugf(template, args...) }
func (z *ZapLogger) Info(args ...interface{})                    { z.sugar.Info(args...) }
func (z *ZapLogger) Infof(template string, args ...interface{})   { z.sugar.Infof(template, args...) }
func (z *ZapLogger) Warn(args ...interface{})                    { z.sugar.Warn(args...) }
func (z *ZapLogger) Warnf(template string, args ...interface{})   { z.sugar.Warnf(template, args...) }
func (z *ZapLogger) Error(args ...interface{})                   { z.sugar.Error(args...) }
func (z *ZapLogger) Errorf(template string, args ...interface{})  { z.sugar.Errorf(template, args...) }
func (z *ZapLogger) Fatal(args ...interface{})                   { z.sugar.Fatal(args...) }
func (z *ZapLogger) Fatalf(template string, args ...interface{})  { z.sugar.Fatalf(template, args...) }
func (z *ZapLogger) Panic(args ...interface{})                   { z.sugar.Panic(args...) }
func (z *ZapLogger) Panicf(template string, args ...interface{})  { z.sugar.Panicf(template, args...) }

// QuietLogger implements Logger by discarding all log messages.
// Fatal and Fatalf still exit the process; Panic and Panicf still panic.
type QuietLogger struct{}

func (n *QuietLogger) Debug(args ...interface{})                   {}
func (n *QuietLogger) Debugf(template string, args ...interface{}) {}
func (n *QuietLogger) Info(args ...interface{})                    {}
func (n *QuietLogger) Infof(template string, args ...interface{})  {}
func (n *QuietLogger) Warn(args ...interface{})                    {}
func (n *QuietLogger) Warnf(template string, args ...interface{})  {}
func (n *QuietLogger) Error(args ...interface{})                    {}
func (n *QuietLogger) Errorf(template string, args ...interface{}) {}
func (n *QuietLogger) Fatal(args ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprint(args...))
	os.Exit(1)
}
func (n *QuietLogger) Fatalf(template string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, template+"\n", args...)
	os.Exit(1)
}
func (n *QuietLogger) Panic(args ...interface{})                   { panic(fmt.Sprint(args...)) }
func (n *QuietLogger) Panicf(template string, args ...interface{}) { panic(fmt.Sprintf(template, args...)) }

// TestLogger implements Logger and panics on Fatal/Panic calls instead of
// calling os.Exit. All log output is discarded to keep test output clean.
type TestLogger struct{}

func (t *TestLogger) Debug(args ...interface{})                   {}
func (t *TestLogger) Debugf(template string, args ...interface{}) {}
func (t *TestLogger) Info(args ...interface{})                    {}
func (t *TestLogger) Infof(template string, args ...interface{})  {}
func (t *TestLogger) Warn(args ...interface{})                    {}
func (t *TestLogger) Warnf(template string, args ...interface{})  {}
func (t *TestLogger) Error(args ...interface{})                   {}
func (t *TestLogger) Errorf(template string, args ...interface{}) {}
func (t *TestLogger) Fatal(args ...interface{}) {
	panic("FATAL: " + fmt.Sprint(args...))
}
func (t *TestLogger) Fatalf(template string, args ...interface{}) {
	panic(fmt.Sprintf("FATAL: "+template, args...))
}
func (t *TestLogger) Panic(args ...interface{}) {
	panic("PANIC: " + fmt.Sprint(args...))
}
func (t *TestLogger) Panicf(template string, args ...interface{}) {
	panic(fmt.Sprintf("PANIC: "+template, args...))
}

// CheckFatal calls L.Fatalf if err is non-nil.
func CheckFatal(err error) {
	if err != nil {
		L.Fatalf("Something went wrong: %q", err)
	}
}
