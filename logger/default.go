package logger

import (
	"fmt"
	"log"
	"os"
)

var _ Logger = (*localLogger)(nil)

// SetDefaultLogger sets the default logger.
// This is not concurrency safe, which means it should only be called during init.
func SetDefaultLogger(l Logger) {
	if l == nil {
		panic("logger must not be nil")
	}
	defaultLogger = l
}

var defaultLogger Logger = &localLogger{
	logger: log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
}

type localLogger struct {
	logger *log.Logger
}

func (ll *localLogger) logf(lv Level, format *string, v ...any) {
	if level > lv {
		return
	}
	msg := lv.String() + fmt.Sprintf(*format, v...)
	ll.logger.Output(4, msg)
}

func (ll *localLogger) Debug(format string, v ...any) {
	ll.logf(LevelDebug, &format, v...)
}

func (ll *localLogger) Info(format string, v ...any) {
	ll.logf(LevelInfo, &format, v...)
}

func (ll *localLogger) Warn(format string, v ...any) {
	ll.logf(LevelWarn, &format, v...)
}

func (ll *localLogger) Error(format string, v ...any) {
	ll.logf(LevelError, &format, v...)
}
