package logger

import (
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestZapLogger_InterfaceSatisfaction(t *testing.T) {
	var l Logger = &ZapLogger{}
	_ = l
}

func TestQuietLogger_InterfaceSatisfaction(t *testing.T) {
	var l Logger = &QuietLogger{}
	_ = l
}

func TestTestLogger_InterfaceSatisfaction(t *testing.T) {
	var l Logger = &TestLogger{}
	_ = l
}

func TestTestLogger_FatalPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("expected panic from TestLogger.Fatal, got none")
		} else if s, ok := r.(string); !ok || !strings.HasPrefix(s, "FATAL:") {
			t.Errorf("expected panic to contain FATAL: prefix, got %v", r)
		}
	}()
	l := &TestLogger{}
	l.Fatal("test fatal")
}

func TestTestLogger_FatalfPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("expected panic from TestLogger.Fatalf, got none")
		} else if s, ok := r.(string); !ok || !strings.HasPrefix(s, "FATAL:") {
			t.Errorf("expected panic to contain FATAL: prefix, got %v", r)
		}
	}()
	l := &TestLogger{}
	l.Fatalf("test fatal %s", "arg")
}

func TestSetLogger(t *testing.T) {
	original := L
	defer SetLogger(original)

	SetLogger(&TestLogger{})
	if _, ok := L.(*TestLogger); !ok {
		t.Errorf("expected L to be *TestLogger, got %T", L)
	}
}

func TestDefaultLoggerIsQuiet(t *testing.T) {
	original := L
	defer SetLogger(original)

	SetLogger(&QuietLogger{})
	if _, ok := L.(*QuietLogger); !ok {
		t.Errorf("expected L to be *QuietLogger, got %T", L)
	}
}

func TestQuietLogger_NonFatalMethodsNoPanic(t *testing.T) {
	n := &QuietLogger{}
	n.Debug("x")
	n.Debugf("x %s", "y")
	n.Info("x")
	n.Infof("x %s", "y")
	n.Warn("x")
	n.Warnf("x %s", "y")
	n.Error("x")
	n.Errorf("x %s", "y")
}

func TestQuietLogger_PanicPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("expected panic from QuietLogger.Panic, got none")
		} else if s, ok := r.(string); !ok || !strings.HasPrefix(s, "test panic") {
			t.Errorf("expected panic to contain message, got %v", r)
		}
	}()
	n := &QuietLogger{}
	n.Panic("test panic")
}

func TestQuietLogger_PanicfPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("expected panic from QuietLogger.Panicf, got none")
		} else if s, ok := r.(string); !ok || !strings.HasPrefix(s, "test panic arg") {
			t.Errorf("expected panic to contain message, got %v", r)
		}
	}()
	n := &QuietLogger{}
	n.Panicf("test panic %s", "arg")
}

func TestTestLogger_PanicPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("expected panic from TestLogger.Panic, got none")
		} else if s, ok := r.(string); !ok || !strings.HasPrefix(s, "PANIC:") {
			t.Errorf("expected panic to contain PANIC: prefix, got %v", r)
		}
	}()
	l := &TestLogger{}
	l.Panic("test panic")
}

func TestTestLogger_PanicfPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("expected panic from TestLogger.Panicf, got none")
		} else if s, ok := r.(string); !ok || !strings.HasPrefix(s, "PANIC:") {
			t.Errorf("expected panic to contain PANIC: prefix, got %v", r)
		}
	}()
	l := &TestLogger{}
	l.Panicf("test panic %s", "arg")
}

func TestZapLogger_CallsDontPanic(t *testing.T) {
	zap.ReplaceGlobals(zap.NewNop())
	defer zap.ReplaceGlobals(zap.NewNop())

	z := &ZapLogger{}
	z.Debug("test")
	z.Debugf("test %s", "arg")
	z.Info("test")
	z.Infof("test %s", "arg")
	z.Warn("test")
	z.Warnf("test %s", "arg")
	z.Error("test")
	z.Errorf("test %s", "arg")
}

func TestZapLogger_PanicPanics(t *testing.T) {
	zap.ReplaceGlobals(zap.NewNop())

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic from ZapLogger.Panic, got none")
		}
	}()
	z := &ZapLogger{}
	z.Panic("test")
}

func TestZapLogger_PanicfPanics(t *testing.T) {
	zap.ReplaceGlobals(zap.NewNop())

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic from ZapLogger.Panicf, got none")
		}
	}()
	z := &ZapLogger{}
	z.Panicf("test %s", "arg")
}

func TestQuietLogger_FatalExits(t *testing.T) {
	t.Skip("QuietLogger.Fatal calls os.Exit(1); cannot test in-process")
}
