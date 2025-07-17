package watcher

import "github.com/douhashi/osoba/internal/logger"

// mockLogger はテスト用のモックロガー
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, keysAndValues ...interface{}) {}
func (m *mockLogger) Info(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Warn(msg string, keysAndValues ...interface{})  {}
func (m *mockLogger) Error(msg string, keysAndValues ...interface{}) {}
func (m *mockLogger) WithFields(keysAndValues ...interface{}) logger.Logger {
	return m
}

// NewMockLogger はテスト用のモックロガーを作成
func NewMockLogger() logger.Logger {
	return &mockLogger{}
}
