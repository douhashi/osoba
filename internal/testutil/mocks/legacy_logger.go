package mocks

import (
	"github.com/douhashi/osoba/internal/logger"
	"github.com/stretchr/testify/mock"
)

// MockLegacyLogger is a mock implementation of logger.Logger interface (legacy)
// This is for backward compatibility with the old logger interface
type MockLegacyLogger struct {
	mock.Mock
}

// NewMockLegacyLogger creates a new instance of MockLegacyLogger
func NewMockLegacyLogger() *MockLegacyLogger {
	return &MockLegacyLogger{}
}

// WithDefaultBehavior sets up common default behaviors for the mock
func (m *MockLegacyLogger) WithDefaultBehavior() *MockLegacyLogger {
	// ログメソッドのデフォルト動作（何もしない）
	m.On("Debug", mock.Anything, mock.Anything).Maybe().Return()
	m.On("Info", mock.Anything, mock.Anything).Maybe().Return()
	m.On("Warn", mock.Anything, mock.Anything).Maybe().Return()
	m.On("Error", mock.Anything, mock.Anything).Maybe().Return()

	// WithFieldsは自分自身を返す
	m.On("WithFields", mock.Anything).Maybe().Return(m)

	return m
}

// Debug mocks the Debug method
func (m *MockLegacyLogger) Debug(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues)
	m.Called(args...)
}

// Info mocks the Info method
func (m *MockLegacyLogger) Info(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues)
	m.Called(args...)
}

// Warn mocks the Warn method
func (m *MockLegacyLogger) Warn(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues)
	m.Called(args...)
}

// Error mocks the Error method
func (m *MockLegacyLogger) Error(msg string, keysAndValues ...interface{}) {
	args := []interface{}{msg}
	args = append(args, keysAndValues)
	m.Called(args...)
}

// WithFields mocks the WithFields method
func (m *MockLegacyLogger) WithFields(keysAndValues ...interface{}) logger.Logger {
	args := m.Called(keysAndValues)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(logger.Logger)
}

// Ensure MockLegacyLogger implements logger.Logger interface
var _ logger.Logger = (*MockLegacyLogger)(nil)
