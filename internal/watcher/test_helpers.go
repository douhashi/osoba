package watcher

import (
	"sync"

	"github.com/douhashi/osoba/internal/logger"
	"github.com/douhashi/osoba/internal/testutil/mocks"
)

// MockLogEntry represents a single log entry for testing
type MockLogEntry struct {
	Level   string
	Message string
	Fields  []interface{}
}

// mockLogger はテスト用のモックロガー（互換性のため残す）
type mockLogger struct {
	*mocks.MockLegacyLogger
	mu   sync.Mutex
	logs []MockLogEntry
}

// GetLogs returns all logged entries
func (m *mockLogger) GetLogs() []MockLogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]MockLogEntry, len(m.logs))
	copy(result, m.logs)
	return result
}

// Debug records debug log
func (m *mockLogger) Debug(msg string, keysAndValues ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, MockLogEntry{
		Level:   "DEBUG",
		Message: msg,
		Fields:  keysAndValues,
	})
	m.MockLegacyLogger.Debug(msg, keysAndValues...)
}

// Info records info log
func (m *mockLogger) Info(msg string, keysAndValues ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, MockLogEntry{
		Level:   "INFO",
		Message: msg,
		Fields:  keysAndValues,
	})
	m.MockLegacyLogger.Info(msg, keysAndValues...)
}

// Warn records warn log
func (m *mockLogger) Warn(msg string, keysAndValues ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, MockLogEntry{
		Level:   "WARN",
		Message: msg,
		Fields:  keysAndValues,
	})
	m.MockLegacyLogger.Warn(msg, keysAndValues...)
}

// Error records error log
func (m *mockLogger) Error(msg string, keysAndValues ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, MockLogEntry{
		Level:   "ERROR",
		Message: msg,
		Fields:  keysAndValues,
	})
	m.MockLegacyLogger.Error(msg, keysAndValues...)
}

// WithFields mocks the WithFields method
func (m *mockLogger) WithFields(keysAndValues ...interface{}) logger.Logger {
	return m
}

// NewMockLogger はテスト用のモックロガーを作成
func NewMockLogger() logger.Logger {
	return &mockLogger{
		MockLegacyLogger: mocks.NewMockLegacyLogger().WithDefaultBehavior(),
		logs:             make([]MockLogEntry, 0),
	}
}
