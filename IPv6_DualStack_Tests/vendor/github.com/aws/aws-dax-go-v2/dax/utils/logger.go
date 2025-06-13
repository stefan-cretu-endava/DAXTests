package utils

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/smithy-go/logging"
)

// A LogLevelType defines the level logging should be performed at. Used to instruct
// the SDK which statements should be logged.
type LogLevelType uint

// Value returns the LogLevel value or the default value LogOff if the LogLevel
// is nil. Safe to use on nil value LogLevelTypes.
func (l *LogLevelType) Value() LogLevelType {
	if l != nil {
		return *l
	}
	return LogOff
}

// Matches returns true if the v LogLevel is enabled by this LogLevel. Should be
// used with logging sub levels. Is safe to use on nil value LogLevelTypes. If
// LogLevel is nil, will default to LogOff comparison.
func (l *LogLevelType) Matches(v LogLevelType) bool {
	c := l.Value()
	return c&v == v
}

// AtLeast returns true if this LogLevel is at least high enough to satisfies v.
// Is safe to use on nil value LogLevelTypes. If LogLevel is nil, will default
// to LogOff comparison.
func (l *LogLevelType) AtLeast(v LogLevelType) bool {
	c := l.Value()
	return c >= v
}

const (
	// Default log for info, warn and error message
	LogOff LogLevelType = 0

	// LogDebug state that debug output should be logged by the SDK. This should
	// be used to inspect request made and responses received.
	LogDebug LogLevelType = 1

	// LogDebugWithRequestRetries states the SDK should log when service requests will
	// be retried. This should be used to log when you want to log when service
	// requests are being retried. Will also enable LogDebug.
	LogDebugWithRequestRetries LogLevelType = 2
)

// Create a default logger implementing smithy-go logging.Logger interface
type defaultLogger struct {
	logger *log.Logger
}

type Logger interface {
	Logf(logging.Classification, string, ...interface{})
}

// NewDefaultLogger returns a Logger which will write log messages to stdout, and
// use same formatting runes as the stdlib log.Logger
func NewDefaultLogger() Logger {
	return &defaultLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

func (l *defaultLogger) Logf(classification logging.Classification, format string, v ...interface{}) {
	// Get the current time in RFC3339 format
	// currentTime := time.Now().Format("2006/01/02 15:04:05")

	// Print the log with the timestamp, classification, and formatted message
	l.logger.Print(fmt.Sprintf("[%s] %s\n", classification, fmt.Sprintf(format, v...)))
}
