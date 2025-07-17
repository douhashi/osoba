package watcher

import (
	"sync"

	"github.com/douhashi/osoba/internal/logger"
)

// MockLogEntry represents a single log entry for testing
type MockLogEntry struct {
	Level   string
	Message string
	Fields  []interface{}
}

// mockLogger はテスト用のモックロガー
type mockLogger struct {
	mu   sync.Mutex
	logs []MockLogEntry
}

func (m *mockLogger) Debug(msg string, keysAndValues ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, MockLogEntry{
		Level:   "DEBUG",
		Message: msg,
		Fields:  keysAndValues,
	})
}

func (m *mockLogger) Info(msg string, keysAndValues ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, MockLogEntry{
		Level:   "INFO",
		Message: msg,
		Fields:  keysAndValues,
	})
}

func (m *mockLogger) Warn(msg string, keysAndValues ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, MockLogEntry{
		Level:   "WARN",
		Message: msg,
		Fields:  keysAndValues,
	})
}

func (m *mockLogger) Error(msg string, keysAndValues ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, MockLogEntry{
		Level:   "ERROR",
		Message: msg,
		Fields:  keysAndValues,
	})
}

func (m *mockLogger) WithFields(keysAndValues ...interface{}) logger.Logger {
	return m
}

// GetLogs returns all logged entries
func (m *mockLogger) GetLogs() []MockLogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]MockLogEntry, len(m.logs))
	copy(result, m.logs)
	return result
}

// NewMockLogger はテスト用のモックロガーを作成
func NewMockLogger() logger.Logger {
	return &mockLogger{
		logs: make([]MockLogEntry, 0),
	}
}
