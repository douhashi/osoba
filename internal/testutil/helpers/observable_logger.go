package helpers

import (
	"github.com/douhashi/osoba/internal/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// ObservableLogger is a logger implementation that captures logs for testing
type ObservableLogger struct {
	sugar    *zap.SugaredLogger
	core     zapcore.Core
	recorded *observer.ObservedLogs
}

// NewObservableLogger creates a new observable logger
func NewObservableLogger(level zapcore.Level) (*ObservableLogger, *observer.ObservedLogs) {
	core, recorded := observer.New(level)
	sugar := zap.New(core).Sugar()
	return &ObservableLogger{
		sugar:    sugar,
		core:     core,
		recorded: recorded,
	}, recorded
}

// Debug logs a debug message
func (l *ObservableLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.sugar.Debugw(msg, keysAndValues...)
}

// Info logs an info message
func (l *ObservableLogger) Info(msg string, keysAndValues ...interface{}) {
	l.sugar.Infow(msg, keysAndValues...)
}

// Warn logs a warning message
func (l *ObservableLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.sugar.Warnw(msg, keysAndValues...)
}

// Error logs an error message
func (l *ObservableLogger) Error(msg string, keysAndValues ...interface{}) {
	l.sugar.Errorw(msg, keysAndValues...)
}

// WithFields returns a new logger with additional fields
func (l *ObservableLogger) WithFields(keysAndValues ...interface{}) logger.Logger {
	return &ObservableLogger{
		sugar:    l.sugar.With(keysAndValues...),
		core:     l.core,
		recorded: l.recorded,
	}
}

// Ensure ObservableLogger implements logger.Logger interface
var _ logger.Logger = (*ObservableLogger)(nil)
