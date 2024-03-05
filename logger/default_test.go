package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testLogger struct{}

func TestSetDefaultLogger(t *testing.T) {
	tl := &testLogger{}
	SetDefaultLogger(tl)

	assert.Equal(t, tl, defaultLogger)
}

func (l *testLogger) Debug(format string, v ...any) {}
func (l *testLogger) Info(format string, v ...any)  {}
func (l *testLogger) Warn(format string, v ...any)  {}
func (l *testLogger) Error(format string, v ...any) {}
