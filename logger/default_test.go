package logger

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"testing"
)

func TestLocalLogger(t *testing.T) {
	t.Run("TestDebug", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		ll := &localLogger{
			logger: log.New(buffer, "", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
		}
		ll.Debug("debug message")
		if got, want := buffer.String(), "[DEBUG] debug message\n"; !strings.Contains(got, want) {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("TestInfo", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		ll := &localLogger{
			logger: log.New(buffer, "", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
		}
		ll.Info("info message")
		if got, want := buffer.String(), "[INFO] info message\n"; !strings.Contains(got, want) {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("TestWarn", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		ll := &localLogger{
			logger: log.New(buffer, "", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
		}
		ll.Warn("warn message")
		if got, want := buffer.String(), "[WARN] warn message\n"; !strings.Contains(got, want) {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("TestError", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		ll := &localLogger{
			logger: log.New(buffer, "", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
		}
		ll.Error("error message")
		if got, want := buffer.String(), "[ERROR] error message\n"; !strings.Contains(got, want) {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("TestLogf", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		ll := &localLogger{
			logger: log.New(buffer, "", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
		}
		testCases := []struct {
			level     Level
			format    string
			expected  string
			shouldLog bool
		}{
			{LevelDebug, "debug message %s", "[DEBUG] debug message test\n", true},
			{LevelInfo, "info message %s", "[INFO] info message test\n", true},
			{LevelWarn, "warn message %s", "[WARN] warn message test\n", true},
			{LevelError, "error message %s", "[ERROR] error message test\n", true},
		}

		for _, testCase := range testCases {
			t.Run(fmt.Sprintf("Level%s", testCase.level), func(t *testing.T) {
				buffer.Reset()
				ll.logf(testCase.level, &testCase.format, "test")
				if testCase.shouldLog {
					if !strings.Contains(buffer.String(), testCase.expected) {
						t.Errorf("Expected output: %s, but got: %s", testCase.expected, buffer.String())
					}
				} else {
					if buffer.String() != "" {
						t.Errorf("Expected no output, but got: %s", buffer.String())
					}
				}
			})
		}
	})
}
