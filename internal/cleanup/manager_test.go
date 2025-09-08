package cleanup

import (
	"context"
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockLogger はテスト用のロガー実装
type mockLogger struct {
	mock.Mock
}

func (m *mockLogger) Debug(msg string, args ...interface{}) {
	m.Called(msg, args)
}

func (m *mockLogger) Info(msg string, args ...interface{}) {
	m.Called(msg, args)
}

func (m *mockLogger) Warn(msg string, args ...interface{}) {
	m.Called(msg, args)
}

func (m *mockLogger) Error(msg string, args ...interface{}) {
	m.Called(msg, args)
}

func (m *mockLogger) WithFields(fields ...interface{}) logger.Logger {
	args := m.Called(fields)
	return args.Get(0).(logger.Logger)
}

// mockCommandExecutor はtmuxコマンド実行のモック
type mockCommandExecutor struct {
	mock.Mock
}

func (m *mockCommandExecutor) Execute(cmd string, args ...string) (string, error) {
	callArgs := m.Called(cmd, args)
	return callArgs.String(0), callArgs.Error(1)
}

func TestNewManager(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		logger      logger.Logger
		wantNil     bool
	}{
		{
			name:        "with session name and logger",
			sessionName: "test-session",
			logger:      &mockLogger{},
			wantNil:     false,
		},
		{
			name:        "with empty session name",
			sessionName: "",
			logger:      &mockLogger{},
			wantNil:     false,
		},
		{
			name:        "without logger",
			sessionName: "test-session",
			logger:      nil,
			wantNil:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(tt.sessionName, tt.logger)
			if tt.wantNil {
				assert.Nil(t, manager)
			} else {
				assert.NotNil(t, manager)
			}
		})
	}
}

func TestCleanupIssueResources_Success(t *testing.T) {
	// テスト用のモックを準備
	mockLog := &mockLogger{}
	mockExecutor := &mockCommandExecutor{}

	// tmuxウィンドウの存在確認と削除が成功するケース
	mockLog.On("Debug", mock.Anything, mock.Anything).Return()
	mockLog.On("Info", mock.Anything, mock.Anything).Return()
	mockLog.On("Warn", mock.Anything, mock.Anything).Return() // worktree削除のWarning用

	// tmuxコマンドのモック設定
	// ListWindowsForIssue相当の処理をモック（Issue番号にマッチするウィンドウを返す）
	mockExecutor.On("Execute", "tmux", []string{"list-windows", "-t", "test-session", "-F", "#{window_index}:#{window_name}:#{window_active}:#{window_panes}"}).
		Return("0:issue-123:1:1\n1:123-plan:0:1\n2:other-window:0:1", nil)

	// KillWindow相当の処理をモック（各ウィンドウに対して）
	mockExecutor.On("Execute", "tmux", []string{"kill-window", "-t", "test-session:issue-123"}).
		Return("", nil)
	mockExecutor.On("Execute", "tmux", []string{"kill-window", "-t", "test-session:123-plan"}).
		Return("", nil)

	ctx := context.Background()
	manager := &DefaultManager{
		sessionName: "test-session",
		logger:      mockLog,
		executor:    mockExecutor,
	}

	err := manager.CleanupIssueResources(ctx, 123)
	assert.NoError(t, err)

	// モックが期待通り呼ばれたか確認
	mockExecutor.AssertExpectations(t)
}

func TestCleanupIssueResources_NoWindow(t *testing.T) {
	// ウィンドウが存在しない場合のテスト
	mockLog := &mockLogger{}
	mockExecutor := &mockCommandExecutor{}

	mockLog.On("Debug", mock.Anything, mock.Anything).Return()
	mockLog.On("Info", mock.Anything, mock.Anything).Return()
	mockLog.On("Warn", mock.Anything, mock.Anything).Return() // worktree削除のWarning用

	// ウィンドウが見つからない場合
	mockExecutor.On("Execute", "tmux", []string{"list-windows", "-t", "test-session", "-F", "#{window_index}:#{window_name}:#{window_active}:#{window_panes}"}).
		Return("0:other-window:1:1", nil)

	ctx := context.Background()
	manager := &DefaultManager{
		sessionName: "test-session",
		logger:      mockLog,
		executor:    mockExecutor,
	}

	err := manager.CleanupIssueResources(ctx, 123)
	// ウィンドウが存在しなくてもエラーにはしない
	assert.NoError(t, err)

	mockExecutor.AssertExpectations(t)
}

func TestCleanupIssueResources_TmuxError(t *testing.T) {
	// tmuxコマンドがエラーを返す場合のテスト
	mockLog := &mockLogger{}
	mockExecutor := &mockCommandExecutor{}

	mockLog.On("Warn", mock.Anything, mock.Anything).Return()
	mockLog.On("Debug", mock.Anything, mock.Anything).Return()

	// tmuxコマンドがエラーを返す
	mockExecutor.On("Execute", "tmux", []string{"list-windows", "-t", "test-session", "-F", "#{window_index}:#{window_name}:#{window_active}:#{window_panes}"}).
		Return("", errors.New("tmux server not running"))

	ctx := context.Background()
	manager := &DefaultManager{
		sessionName: "test-session",
		logger:      mockLog,
		executor:    mockExecutor,
	}

	err := manager.CleanupIssueResources(ctx, 123)
	// エラーが発生してもクリーンアップ処理は継続（エラーを返さない）
	assert.NoError(t, err)

	mockExecutor.AssertExpectations(t)
	mockLog.AssertExpectations(t)
}

func TestCleanupIssueResources_NoSessionName(t *testing.T) {
	// セッション名が指定されていない場合のテスト
	mockLog := &mockLogger{}

	mockLog.On("Warn", mock.Anything, mock.Anything).Return()

	ctx := context.Background()
	manager := &DefaultManager{
		sessionName: "", // セッション名が空
		logger:      mockLog,
	}

	err := manager.CleanupIssueResources(ctx, 123)
	// セッション名が空でもエラーにはしない（警告ログを出力して継続）
	assert.NoError(t, err)

	mockLog.AssertExpectations(t)
}

func TestCleanupIssueResources_MultipleWindows(t *testing.T) {
	// 複数のウィンドウが存在する場合のテスト（フェーズごとのウィンドウ）
	mockLog := &mockLogger{}
	mockExecutor := &mockCommandExecutor{}

	mockLog.On("Debug", mock.Anything, mock.Anything).Return()
	mockLog.On("Info", mock.Anything, mock.Anything).Return()
	mockLog.On("Warn", mock.Anything, mock.Anything).Return() // worktree削除のWarning用

	// 複数のウィンドウが存在（123-plan, 123-implement, 123-review）
	mockExecutor.On("Execute", "tmux", []string{"list-windows", "-t", "test-session", "-F", "#{window_index}:#{window_name}:#{window_active}:#{window_panes}"}).
		Return("0:123-plan:0:1\n1:123-implement:0:1\n2:123-review:1:1\n3:other-window:0:1", nil)

	// 各ウィンドウの削除
	mockExecutor.On("Execute", "tmux", []string{"kill-window", "-t", "test-session:123-plan"}).
		Return("", nil)
	mockExecutor.On("Execute", "tmux", []string{"kill-window", "-t", "test-session:123-implement"}).
		Return("", nil)
	mockExecutor.On("Execute", "tmux", []string{"kill-window", "-t", "test-session:123-review"}).
		Return("", nil)

	ctx := context.Background()
	manager := &DefaultManager{
		sessionName: "test-session",
		logger:      mockLog,
		executor:    mockExecutor,
	}

	err := manager.CleanupIssueResources(ctx, 123)
	assert.NoError(t, err)

	mockExecutor.AssertExpectations(t)
}

