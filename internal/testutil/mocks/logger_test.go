package mocks_test

import (
	"testing"

	"github.com/douhashi/osoba/internal/log"
	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMockLogger_BasicLogging(t *testing.T) {
	tests := []struct {
		name    string
		logFunc func(log.Logger, string)
		method  string
		message string
	}{
		{
			name:    "debug log",
			logFunc: func(l log.Logger, msg string) { l.Debug(msg) },
			method:  "Debug",
			message: "debug message",
		},
		{
			name:    "info log",
			logFunc: func(l log.Logger, msg string) { l.Info(msg) },
			method:  "Info",
			message: "info message",
		},
		{
			name:    "warn log",
			logFunc: func(l log.Logger, msg string) { l.Warn(msg) },
			method:  "Warn",
			message: "warn message",
		},
		{
			name:    "error log",
			logFunc: func(l log.Logger, msg string) { l.Error(msg) },
			method:  "Error",
			message: "error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := mocks.NewMockLogger()
			mockLogger.On(tt.method, tt.message).Return()

			tt.logFunc(mockLogger, tt.message)

			mockLogger.AssertExpectations(t)
		})
	}
}

func TestMockLogger_WithField(t *testing.T) {
	mockLogger := mocks.NewMockLogger()
	childLogger := mocks.NewMockLogger()

	mockLogger.On("WithField", "key", "value").Return(childLogger)
	childLogger.On("Info", "test message").Return()

	result := mockLogger.WithField("key", "value")
	result.Info("test message")

	assert.Equal(t, childLogger, result)
	mockLogger.AssertExpectations(t)
	childLogger.AssertExpectations(t)
}

func TestMockLogger_WithFields(t *testing.T) {
	mockLogger := mocks.NewMockLogger()
	childLogger := mocks.NewMockLogger()

	fields := log.Fields{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	mockLogger.On("WithFields", fields).Return(childLogger)
	childLogger.On("Error", "error occurred").Return()

	result := mockLogger.WithFields(fields)
	result.Error("error occurred")

	assert.Equal(t, childLogger, result)
	mockLogger.AssertExpectations(t)
	childLogger.AssertExpectations(t)
}

func TestMockLogger_WithComponent(t *testing.T) {
	mockLogger := mocks.NewMockLogger()
	childLogger := mocks.NewMockLogger()

	mockLogger.On("WithComponent", "watcher").Return(childLogger)
	childLogger.On("Debug", "component initialized").Return()

	result := mockLogger.WithComponent("watcher")
	result.Debug("component initialized")

	assert.Equal(t, childLogger, result)
	mockLogger.AssertExpectations(t)
	childLogger.AssertExpectations(t)
}

func TestMockLogger_WithDefaultBehavior(t *testing.T) {
	mockLogger := mocks.NewMockLogger().WithDefaultBehavior()

	// デフォルト動作のテスト - 何も返さないが呼び出しは記録される
	mockLogger.Debug("debug message")
	mockLogger.Info("info message")
	mockLogger.Warn("warn message")
	mockLogger.Error("error message")

	// WithFieldは自分自身を返す
	result := mockLogger.WithField("key", "value")
	assert.Equal(t, mockLogger, result)

	// WithFieldsも自分自身を返す
	result = mockLogger.WithFields(log.Fields{"key": "value"})
	assert.Equal(t, mockLogger, result)

	// WithComponentも自分自身を返す
	result = mockLogger.WithComponent("test")
	assert.Equal(t, mockLogger, result)
}

func TestMockLogger_ChainedCalls(t *testing.T) {
	mockLogger := mocks.NewMockLogger()
	withFieldLogger := mocks.NewMockLogger()
	withComponentLogger := mocks.NewMockLogger()

	// チェーンされた呼び出しのモック
	mockLogger.On("WithField", "request_id", "123").Return(withFieldLogger)
	withFieldLogger.On("WithComponent", "api").Return(withComponentLogger)
	withComponentLogger.On("Info", "processing request").Return()

	// 実際の使用例をシミュレート
	mockLogger.
		WithField("request_id", "123").
		WithComponent("api").
		Info("processing request")

	mockLogger.AssertExpectations(t)
	withFieldLogger.AssertExpectations(t)
	withComponentLogger.AssertExpectations(t)
}

func TestMockLogger_ComplexScenario(t *testing.T) {
	mockLogger := mocks.NewMockLogger()

	// 複数のログレベルでの呼び出しを期待
	mockLogger.On("Debug", mock.MatchedBy(func(msg string) bool {
		return len(msg) > 0
	})).Times(2)

	mockLogger.On("Info", mock.Anything).Once()
	mockLogger.On("Error", mock.Anything).Once()

	// 実行
	mockLogger.Debug("first debug")
	mockLogger.Debug("second debug")
	mockLogger.Info("info message")
	mockLogger.Error("error message")

	mockLogger.AssertExpectations(t)
}
