package actions

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/douhashi/osoba/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// testZapLogger はテスト用のlogger実装
type testZapLogger struct {
	sugar *zap.SugaredLogger
}

func (l *testZapLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.sugar.Debugw(msg, keysAndValues...)
}

func (l *testZapLogger) Info(msg string, keysAndValues ...interface{}) {
	l.sugar.Infow(msg, keysAndValues...)
}

func (l *testZapLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.sugar.Warnw(msg, keysAndValues...)
}

func (l *testZapLogger) Error(msg string, keysAndValues ...interface{}) {
	l.sugar.Errorw(msg, keysAndValues...)
}

func (l *testZapLogger) WithFields(keysAndValues ...interface{}) logger.Logger {
	return &testZapLogger{
		sugar: l.sugar.With(keysAndValues...),
	}
}

// createTestLogger はテスト用のloggerを作成し、出力をキャプチャするバッファも返す
func createTestLogger() (logger.Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	encoder := zapcore.NewJSONEncoder(encoderConfig)
	core := zapcore.NewCore(encoder, zapcore.AddSync(buf), zapcore.DebugLevel)

	zapLogger := zap.New(core, zap.AddCallerSkip(1))
	sugar := zapLogger.Sugar()

	return &testZapLogger{sugar: sugar}, buf
}

func TestPlanActionWithLogger(t *testing.T) {
	t.Run("PlanActionでlog.Printfの代わりにlogger.Infoが使用されること", func(t *testing.T) {
		// Arrange
		testLogger, buf := createTestLogger()
		mockStateManager := &MockStateManager{}
		mockTmuxClient := &MockTmuxClient{}
		mockWorktreeManager := &MockWorktreeManager{}
		mockClaudeExecutor := &MockClaudeExecutor{}

		// 処理が成功する設定
		mockStateManager.On("HasBeenProcessed", int64(123), types.IssueStatePlan).Return(false)
		mockStateManager.On("IsProcessing", int64(123)).Return(false)
		mockStateManager.On("SetState", int64(123), types.IssueStatePlan, types.IssueStatusProcessing).Return()
		mockTmuxClient.On("CreateWindowForIssue", "test-session", 123).Return(nil)
		mockTmuxClient.On("SelectOrCreatePaneForPhase", "test-session", "issue-123", "plan-phase").Return(nil)
		mockWorktreeManager.On("UpdateMainBranch", mock.Anything).Return(nil)
		mockWorktreeManager.On("CreateWorktreeForIssue", mock.Anything, 123).Return(nil)
		mockWorktreeManager.On("GetWorktreePathForIssue", 123).Return("/test/path")
		mockClaudeExecutor.On("ExecuteInTmux", mock.Anything, mock.Anything, mock.Anything, "test-session", "issue-123", "/test/path").Return(nil)
		mockStateManager.On("MarkAsCompleted", int64(123), types.IssueStatePlan).Return()

		claudeConfig := &claude.ClaudeConfig{
			Phases: map[string]*claude.PhaseConfig{
				"plan": {
					Args:   []string{"--test"},
					Prompt: "test template {{issue-number}}",
				},
			},
		}

		// PlanActionWithLoggerを作成（この関数はまだ存在しないため、テストは失敗する）
		action := NewPlanActionWithLogger(
			"test-session",
			mockTmuxClient,
			mockStateManager,
			mockWorktreeManager,
			mockClaudeExecutor,
			claudeConfig,
			testLogger,
		)

		issueNumber := 123
		issue := &github.Issue{
			Number: &issueNumber,
			Title:  stringPtr("Test Issue"),
		}

		// Act
		err := action.Execute(context.Background(), issue)

		// Assert
		assert.NoError(t, err)

		// ログ出力を確認
		logOutput := buf.String()
		assert.Contains(t, logOutput, "Executing plan action")
		assert.Contains(t, logOutput, "issue_number")
		assert.Contains(t, logOutput, "123")

		// 従来のlog.Printfが使用されていないことを確認（標準出力にログが出力されていない）
		// 実際の実装では、log.Printfを使用せずにlogger.Infoを使用することを確認

		mockStateManager.AssertExpectations(t)
		mockTmuxClient.AssertExpectations(t)
		mockWorktreeManager.AssertExpectations(t)
		mockClaudeExecutor.AssertExpectations(t)
	})

	t.Run("ログに構造化された情報が含まれること", func(t *testing.T) {
		// Arrange
		testLogger, buf := createTestLogger()
		mockStateManager := &MockStateManager{}

		// 既に処理済みの場合をテスト
		mockStateManager.On("HasBeenProcessed", int64(456), types.IssueStatePlan).Return(true)

		action := NewPlanActionWithLogger(
			"test-session",
			&MockTmuxClient{},
			mockStateManager,
			&MockWorktreeManager{},
			&MockClaudeExecutor{},
			&claude.ClaudeConfig{},
			testLogger,
		)

		issueNumber := 456
		issue := &github.Issue{
			Number: &issueNumber,
			Title:  stringPtr("Already Processed Issue"),
		}

		// Act
		err := action.Execute(context.Background(), issue)

		// Assert
		assert.NoError(t, err)

		// 構造化ログの確認
		logOutput := buf.String()
		assert.Contains(t, logOutput, "issue_number")
		assert.Contains(t, logOutput, "456")
		assert.Contains(t, logOutput, "already been processed")

		// JSONログ形式であることを確認
		assert.True(t, strings.Contains(logOutput, "{") && strings.Contains(logOutput, "}"))

		mockStateManager.AssertExpectations(t)
	})
}

// stringPtr はテスト用のstring pointer helper
func stringPtr(s string) *string {
	return &s
}
