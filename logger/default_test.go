package logger

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testLogger struct {
	buffer *bytes.Buffer
}

func TestSetDefaultLogger(t *testing.T) {
	tl := &testLogger{}
	SetDefaultLogger(tl)

	assert.Equal(t, tl, defaultLogger)
}

func TestLogger(t *testing.T) {
	t.Run("TestDebug", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		testLogger := &testLogger{buffer: buffer}
		SetDefaultLogger(testLogger)
		Debug("debug message")
		if got, want := buffer.String(), "[DEBUG] debug message\n"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("TestInfo", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		testLogger := &testLogger{buffer: buffer}
		SetDefaultLogger(testLogger)
		Info("info message")
		if got, want := buffer.String(), "[INFO] info message\n"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("TestWarn", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		testLogger := &testLogger{buffer: buffer}
		SetDefaultLogger(testLogger)
		Warn("warn message")
		if got, want := buffer.String(), "[WARN] warn message\n"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("TestError", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		testLogger := &testLogger{buffer: buffer}
		SetDefaultLogger(testLogger)
		Error("error message")
		if got, want := buffer.String(), "[ERROR] error message\n"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("TestLevel", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		testLogger := &testLogger{buffer: buffer}
		SetDefaultLogger(testLogger)
		SetLevel(LevelInfo)
		Debug("debug message")
		Info("info message")
		Warn("warn message")
		Error("error message")
		if got, want := buffer.String(), "[INFO] info message\n[WARN] warn message\n[ERROR] error message\n"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func (tl *testLogger) Debug(format string, v ...any) {
	fmt.Fprintf(tl.buffer, "[DEBUG] "+format+"\n", v...)
}

func (tl *testLogger) Info(format string, v ...any) {
	fmt.Fprintf(tl.buffer, "[INFO] "+format+"\n", v...)
}

func (tl *testLogger) Warn(format string, v ...any) {
	fmt.Fprintf(tl.buffer, "[WARN] "+format+"\n", v...)
}

func (tl *testLogger) Error(format string, v ...any) {
	fmt.Fprintf(tl.buffer, "[ERROR] "+format+"\n", v...)
}
