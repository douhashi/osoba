package actions

import (
	"context"
	"testing"

	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/testutil/builders"
	"github.com/douhashi/osoba/internal/testutil/helpers"
	"github.com/douhashi/osoba/internal/testutil/mocks"
	tmuxpkg "github.com/douhashi/osoba/internal/tmux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zapcore"
)

func TestBaseExecutor_PrepareWorkspace(t *testing.T) {
	tests := []struct {
		name        string
		issue       *github.Issue
		phase       string
		setupMocks  func(*mocks.MockTmuxManager, *mocks.MockGitWorktreeManager)
		want        *WorkspaceInfo
		wantErr     bool
		errContains string
	}{
		{
			name: "新規workspace作成（Window、Worktree、Pane全て新規）",
			issue: builders.NewIssueBuilder().
				WithNumber(123).
				WithTitle("Test Issue").
				Build(),
			phase: "Plan",
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager) {
				// Window存在確認（なし）
				tmux.On("WindowExists", "test-session", "issue-123").Return(false, nil).Once()
				// Window作成
				tmux.On("CreateWindow", "test-session", "issue-123").Return(nil).Once()

				// Worktree存在確認（なし）
				git.On("WorktreeExistsForIssue", mock.Anything, 123).Return(false, nil).Once()
				// Worktree作成
				git.On("CreateWorktreeForIssue", mock.Anything, 123).Return(nil).Once()

				// Pane検索（なし）
				tmux.On("GetPaneByTitle", "test-session", "issue-123", "Plan").
					Return(nil, assert.AnError).Once()
				// Pane 0のタイトル設定（Planフェーズは既存のpane 0を使用）
				tmux.On("SetPaneTitle", "test-session", "issue-123", 0, "Plan").Return(nil).Once()

				// Worktreeパス取得
				git.On("GetWorktreePathForIssue", 123).Return("/test/worktree/issue-123").Once()
			},
			want: &WorkspaceInfo{
				WindowName:   "issue-123",
				WorktreePath: "/test/worktree/issue-123",
				PaneIndex:    0,
				PaneTitle:    "Plan",
			},
			wantErr: false,
		},
		{
			name: "既存workspace使用（Window、Worktree、Pane全て既存）",
			issue: builders.NewIssueBuilder().
				WithNumber(456).
				WithTitle("Test Issue 2").
				Build(),
			phase: "Implementation",
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager) {
				// Window存在確認（あり）
				tmux.On("WindowExists", "test-session", "issue-456").Return(true, nil).Once()

				// Worktree存在確認（あり）
				git.On("WorktreeExistsForIssue", mock.Anything, 456).Return(true, nil).Once()

				// Pane検索（あり）
				tmux.On("GetPaneByTitle", "test-session", "issue-456", "Implementation").
					Return(&tmuxpkg.PaneInfo{Index: 1, Title: "Implementation", Active: false}, nil).Once()
				// 既存pane選択
				tmux.On("SelectPane", "test-session", "issue-456", 1).Return(nil).Once()

				// Worktreeパス取得
				git.On("GetWorktreePathForIssue", 456).Return("/test/worktree/issue-456").Once()
			},
			want: &WorkspaceInfo{
				WindowName:   "issue-456",
				WorktreePath: "/test/worktree/issue-456",
				PaneIndex:    1,
				PaneTitle:    "Implementation",
			},
			wantErr: false,
		},
		{
			name: "新規pane作成（ImplementationフェーズでWindowとWorktreeは既存）",
			issue: builders.NewIssueBuilder().
				WithNumber(789).
				WithTitle("Test Issue 3").
				Build(),
			phase: "Implementation",
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager) {
				// Window存在確認（あり）
				tmux.On("WindowExists", "test-session", "issue-789").Return(true, nil).Once()

				// Worktree存在確認（あり）
				git.On("WorktreeExistsForIssue", mock.Anything, 789).Return(true, nil).Once()

				// Pane検索（なし）
				tmux.On("GetPaneByTitle", "test-session", "issue-789", "Implementation").
					Return(nil, assert.AnError).Once()
				// 新規pane作成
				tmux.On("CreatePane", "test-session", "issue-789", mock.Anything).
					Return(&tmuxpkg.PaneInfo{Index: 1, Title: "Implementation", Active: true}, nil).Once()

				// Worktreeパス取得
				git.On("GetWorktreePathForIssue", 789).Return("/test/worktree/issue-789").Once()
			},
			want: &WorkspaceInfo{
				WindowName:   "issue-789",
				WorktreePath: "/test/worktree/issue-789",
				PaneIndex:    1,
				PaneTitle:    "Implementation",
			},
			wantErr: false,
		},
		{
			name:  "nilのissue",
			issue: nil,
			phase: "Plan",
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager) {
				// 何も呼ばれない
			},
			want:        nil,
			wantErr:     true,
			errContains: "invalid issue",
		},
		{
			name: "Window作成失敗",
			issue: builders.NewIssueBuilder().
				WithNumber(999).
				Build(),
			phase: "Plan",
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager) {
				// Window存在確認（なし）
				tmux.On("WindowExists", "test-session", "issue-999").Return(false, nil).Once()
				// Window作成失敗
				tmux.On("CreateWindow", "test-session", "issue-999").
					Return(assert.AnError).Once()
			},
			want:        nil,
			wantErr:     true,
			errContains: "failed to create window",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックの作成
			logger, _ := helpers.NewObservableLogger(zapcore.InfoLevel)
			tmuxManager := mocks.NewMockTmuxManager()
			worktreeManager := mocks.NewMockGitWorktreeManager()
			claudeExecutor := mocks.NewMockClaudeCommandBuilder()

			// モックの設定
			tt.setupMocks(tmuxManager, worktreeManager)

			// BaseExecutorの作成
			executor := NewBaseExecutor(
				"test-session",
				tmuxManager,
				worktreeManager,
				claudeExecutor,
				logger,
			)

			// テスト実行
			got, err := executor.PrepareWorkspace(context.Background(), tt.issue, tt.phase)

			// アサーション
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			// モックの期待値確認
			tmuxManager.AssertExpectations(t)
			worktreeManager.AssertExpectations(t)
		})
	}
}

func TestBaseExecutor_ExecuteInWorkspace(t *testing.T) {
	tests := []struct {
		name        string
		workspace   *WorkspaceInfo
		command     string
		setupMocks  func(*mocks.MockTmuxManager)
		wantErr     bool
		errContains string
	}{
		{
			name: "コマンド実行成功",
			workspace: &WorkspaceInfo{
				WindowName:   "issue-123",
				WorktreePath: "/test/worktree/issue-123",
				PaneIndex:    0,
				PaneTitle:    "Plan",
			},
			command: "echo 'Hello World'",
			setupMocks: func(tmux *mocks.MockTmuxManager) {
				expectedCmd := "cd /test/worktree/issue-123 && echo 'Hello World'"
				tmux.On("SendKeys", "test-session", "issue-123", expectedCmd).Return(nil).Once()
			},
			wantErr: false,
		},
		{
			name: "SendKeys失敗",
			workspace: &WorkspaceInfo{
				WindowName:   "issue-456",
				WorktreePath: "/test/worktree/issue-456",
				PaneIndex:    1,
				PaneTitle:    "Implementation",
			},
			command: "npm test",
			setupMocks: func(tmux *mocks.MockTmuxManager) {
				expectedCmd := "cd /test/worktree/issue-456 && npm test"
				tmux.On("SendKeys", "test-session", "issue-456", expectedCmd).
					Return(assert.AnError).Once()
			},
			wantErr:     true,
			errContains: "failed to execute command in workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックの作成
			logger, _ := helpers.NewObservableLogger(zapcore.InfoLevel)
			tmuxManager := mocks.NewMockTmuxManager()
			worktreeManager := mocks.NewMockGitWorktreeManager()
			claudeExecutor := mocks.NewMockClaudeCommandBuilder()

			// モックの設定
			tt.setupMocks(tmuxManager)

			// BaseExecutorの作成
			executor := NewBaseExecutor(
				"test-session",
				tmuxManager,
				worktreeManager,
				claudeExecutor,
				logger,
			)

			// テスト実行
			err := executor.ExecuteInWorkspace(tt.workspace, tt.command)

			// アサーション
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			// モックの期待値確認
			tmuxManager.AssertExpectations(t)
		})
	}
}
