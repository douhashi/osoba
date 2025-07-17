package claude

import (
	"testing"

	"github.com/douhashi/osoba/internal/logger"
	"github.com/stretchr/testify/assert"
)

// mockLogger はテスト用のモックロガー
type mockLogger struct {
	debugCalls []logCall
	infoCalls  []logCall
	warnCalls  []logCall
	errorCalls []logCall
}

type logCall struct {
	msg           string
	keysAndValues []interface{}
}

func (m *mockLogger) Debug(msg string, keysAndValues ...interface{}) {
	m.debugCalls = append(m.debugCalls, logCall{msg: msg, keysAndValues: keysAndValues})
}

func (m *mockLogger) Info(msg string, keysAndValues ...interface{}) {
	m.infoCalls = append(m.infoCalls, logCall{msg: msg, keysAndValues: keysAndValues})
}

func (m *mockLogger) Warn(msg string, keysAndValues ...interface{}) {
	m.warnCalls = append(m.warnCalls, logCall{msg: msg, keysAndValues: keysAndValues})
}

func (m *mockLogger) Error(msg string, keysAndValues ...interface{}) {
	m.errorCalls = append(m.errorCalls, logCall{msg: msg, keysAndValues: keysAndValues})
}

func (m *mockLogger) WithFields(keysAndValues ...interface{}) logger.Logger {
	return m
}

// NewMockLogger はテスト用のモックロガーを作成
func newMockLogger() *mockLogger {
	return &mockLogger{
		debugCalls: []logCall{},
		infoCalls:  []logCall{},
		warnCalls:  []logCall{},
		errorCalls: []logCall{},
	}
}

func TestMockLogger(t *testing.T) {
	t.Run("ログメソッドの動作確認", func(t *testing.T) {
		ml := newMockLogger()

		// 各ログレベルをテスト
		ml.Debug("debug message", "key", "value")
		ml.Info("info message", "key", "value")
		ml.Warn("warn message", "key", "value")
		ml.Error("error message", "key", "value")

		// 呼び出しを検証
		assert.Len(t, ml.debugCalls, 1)
		assert.Equal(t, "debug message", ml.debugCalls[0].msg)
		assert.Equal(t, []interface{}{"key", "value"}, ml.debugCalls[0].keysAndValues)

		assert.Len(t, ml.infoCalls, 1)
		assert.Equal(t, "info message", ml.infoCalls[0].msg)

		assert.Len(t, ml.warnCalls, 1)
		assert.Equal(t, "warn message", ml.warnCalls[0].msg)

		assert.Len(t, ml.errorCalls, 1)
		assert.Equal(t, "error message", ml.errorCalls[0].msg)
	})

	t.Run("WithFieldsの動作確認", func(t *testing.T) {
		ml := newMockLogger()

		// WithFieldsは同じインスタンスを返す
		logger2 := ml.WithFields("component", "test")
		assert.Equal(t, ml, logger2)
	})
}
