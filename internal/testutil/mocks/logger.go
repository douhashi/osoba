package mocks

import (
	"github.com/douhashi/osoba/internal/log"
	"github.com/stretchr/testify/mock"
)

// MockLogger is a mock implementation of log.Logger interface
type MockLogger struct {
	mock.Mock
}

// NewMockLogger creates a new instance of MockLogger
func NewMockLogger() *MockLogger {
	return &MockLogger{}
}

// WithDefaultBehavior sets up common default behaviors for the mock
func (m *MockLogger) WithDefaultBehavior() *MockLogger {
	// ログメソッドのデフォルト動作（何もしない）
	m.On("Debug", mock.Anything).Maybe().Return()
	m.On("Info", mock.Anything).Maybe().Return()
	m.On("Warn", mock.Anything).Maybe().Return()
	m.On("Error", mock.Anything).Maybe().Return()

	// With系メソッドは自分自身を返す
	m.On("WithField", mock.Anything, mock.Anything).Maybe().Return(m)
	m.On("WithFields", mock.Anything).Maybe().Return(m)
	m.On("WithComponent", mock.Anything).Maybe().Return(m)

	return m
}

// Debug mocks the Debug method
func (m *MockLogger) Debug(msg string) {
	m.Called(msg)
}

// Info mocks the Info method
func (m *MockLogger) Info(msg string) {
	m.Called(msg)
}

// Warn mocks the Warn method
func (m *MockLogger) Warn(msg string) {
	m.Called(msg)
}

// Error mocks the Error method
func (m *MockLogger) Error(msg string) {
	m.Called(msg)
}

// WithField mocks the WithField method
func (m *MockLogger) WithField(key string, value interface{}) log.Logger {
	args := m.Called(key, value)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(log.Logger)
}

// WithFields mocks the WithFields method
func (m *MockLogger) WithFields(fields log.Fields) log.Logger {
	args := m.Called(fields)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(log.Logger)
}

// WithComponent mocks the WithComponent method
func (m *MockLogger) WithComponent(component string) log.Logger {
	args := m.Called(component)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(log.Logger)
}

// Ensure MockLogger implements log.Logger interface
var _ log.Logger = (*MockLogger)(nil)
