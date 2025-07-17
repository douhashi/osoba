package mocks

import (
	"github.com/douhashi/osoba/internal/logger"
)

// ObservableLegacyLogger is a mock implementation that captures log calls for verification
type ObservableLegacyLogger struct {
	DebugCalls []LogCall
	InfoCalls  []LogCall
	WarnCalls  []LogCall
	ErrorCalls []LogCall
}

// LogCall represents a captured log call
type LogCall struct {
	Msg           string
	KeysAndValues []interface{}
}

// NewObservableLegacyLogger creates a new observable legacy logger
func NewObservableLegacyLogger() *ObservableLegacyLogger {
	return &ObservableLegacyLogger{
		DebugCalls: []LogCall{},
		InfoCalls:  []LogCall{},
		WarnCalls:  []LogCall{},
		ErrorCalls: []LogCall{},
	}
}

// Debug logs a debug message
func (l *ObservableLegacyLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.DebugCalls = append(l.DebugCalls, LogCall{Msg: msg, KeysAndValues: keysAndValues})
}

// Info logs an info message
func (l *ObservableLegacyLogger) Info(msg string, keysAndValues ...interface{}) {
	l.InfoCalls = append(l.InfoCalls, LogCall{Msg: msg, KeysAndValues: keysAndValues})
}

// Warn logs a warning message
func (l *ObservableLegacyLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.WarnCalls = append(l.WarnCalls, LogCall{Msg: msg, KeysAndValues: keysAndValues})
}

// Error logs an error message
func (l *ObservableLegacyLogger) Error(msg string, keysAndValues ...interface{}) {
	l.ErrorCalls = append(l.ErrorCalls, LogCall{Msg: msg, KeysAndValues: keysAndValues})
}

// WithFields returns a new logger with additional fields
func (l *ObservableLegacyLogger) WithFields(keysAndValues ...interface{}) logger.Logger {
	return l
}

// Ensure ObservableLegacyLogger implements logger.Logger interface
var _ logger.Logger = (*ObservableLegacyLogger)(nil)
