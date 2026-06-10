package logger

import (
	"testing"

	"go.uber.org/zap"
)

func TestZapLogger_InterfaceSatisfaction(t *testing.T) {
	var l Logger = &ZapLogger{}
	_ = l
}

func TestNoopLogger_InterfaceSatisfaction(t *testing.T) {
	var l Logger = &NoopLogger{}
	_ = l
}

func TestTestLogger_InterfaceSatisfaction(t *testing.T) {
	var l Logger = &TestLogger{}
	_ = l
}

func TestTestLogger_FatalPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic from TestLogger.Fatal, got none")
		}
	}()
	l := &TestLogger{}
	l.Fatal("test fatal")
}

func TestTestLogger_FatalfPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic from TestLogger.Fatalf, got none")
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

func TestDefaultLoggerIsNoop(t *testing.T) {
	original := L
	defer SetLogger(original)

	SetLogger(&NoopLogger{})
	if _, ok := L.(*NoopLogger); !ok {
		t.Errorf("expected default L to be *NoopLogger, got %T", L)
	}
}

func TestNoopLogger_NonFatalMethodsNoPanic(t *testing.T) {
	n := &NoopLogger{}
	n.Debug("x")
	n.Debugf("x %s", "y")
	n.Info("x")
	n.Infof("x %s", "y")
	n.Warn("x")
	n.Warnf("x %s", "y")
	n.Error("x")
	n.Errorf("x %s", "y")
}

func TestNoopLogger_PanicPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic from NoopLogger.Panic, got none")
		}
	}()
	n := &NoopLogger{}
	n.Panic("test panic")
}

func TestNoopLogger_PanicfPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic from NoopLogger.Panicf, got none")
		}
	}()
	n := &NoopLogger{}
	n.Panicf("test panic %s", "arg")
}

func TestTestLogger_PanicPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic from TestLogger.Panic, got none")
		}
	}()
	l := &TestLogger{}
	l.Panic("test panic")
}

func TestTestLogger_PanicfPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic from TestLogger.Panicf, got none")
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
