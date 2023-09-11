package logger

// Logger is a logger interface that provides logging function with levels.
type Logger interface {
	Debug(format string, v ...interface{})
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}

// Level defines the priority of a log message.
// When a logger is configured with a level, any log message with a lower
// log level (smaller by integer comparison) will not be output.
type Level int

// The levels of logs.
const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// SetLevel sets the level of logs below which logs will not be output.
// The default log level is LevelDebug.
func SetLevel(lv Level) {
	if lv < LevelDebug || lv > LevelError {
		panic("invalid level")
	}
	level = lv
}

// Error calls the default logger's Error method.
func Error(format string, v ...interface{}) {
	if level > LevelError {
		return
	}
	defaultLogger.Error(format, v...)
}

// Warn calls the default logger's Warn method.
func Warn(format string, v ...interface{}) {
	if level > LevelWarn {
		return
	}
	defaultLogger.Warn(format, v...)
}

// Info calls the default logger's Info method.
func Info(format string, v ...interface{}) {
	if level > LevelInfo {
		return
	}
	defaultLogger.Info(format, v...)
}

// Debug calls the default logger's Debug method.
func Debug(format string, v ...interface{}) {
	if level > LevelDebug {
		return
	}
	defaultLogger.Debug(format, v...)
}

var level Level

var levelNames = map[Level]string{
	LevelDebug: "[DEBUG] ",
	LevelInfo:  "[INFO] ",
	LevelWarn:  "[WARN] ",
	LevelError: "[ERROR] ",
}

// String implementation.
func (lv Level) String() string {
	return levelNames[lv]
}
