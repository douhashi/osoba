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
			name: "セッション不在時の自動作成",
			issue: builders.NewIssueBuilder().
				WithNumber(111).
				WithTitle("Test Session Auto Create").
				Build(),
			phase: "Plan",
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager) {
				// セッション存在確認（なし）
				tmux.On("SessionExists", "test-session").Return(false, nil).Once()
				// セッション作成
				tmux.On("EnsureSession", "test-session").Return(nil).Once()

				// Window存在確認（なし）
				tmux.On("WindowExists", "test-session", "issue-111").Return(false, nil).Once()
				// Window作成（新規ウィンドウ判定付き）
				tmux.On("CreateWindowForIssueWithNewWindowDetection", "test-session", 111).
					Return("issue-111", true, nil).Once()

				// Worktree存在確認（なし）
				git.On("WorktreeExistsForIssue", mock.Anything, 111).Return(false, nil).Once()
				// Worktree作成
				git.On("CreateWorktreeForIssue", mock.Anything, 111).Return(nil).Once()

				// Pane検索（なし）
				tmux.On("GetPaneByTitle", "test-session", "issue-111", "Plan").
					Return(nil, assert.AnError).Once()
				// pane-base-index取得
				tmux.On("GetPaneBaseIndex").Return(0, nil).Once()
				// Pane 0のタイトル設定のみ
				tmux.On("SetPaneTitle", "test-session", "issue-111", 0, "Plan").Return(nil).Once()

				// Worktreeパス取得
				git.On("GetWorktreePathForIssue", 111).Return("/test/worktree/issue-111").Once()
			},
			want: &WorkspaceInfo{
				WindowName:   "issue-111",
				WorktreePath: "/test/worktree/issue-111",
				PaneIndex:    0,
				PaneTitle:    "Plan",
			},
			wantErr: false,
		},
		{
			name: "新規workspace作成（Window、Worktree、Pane全て新規）- pane分割なし",
			issue: builders.NewIssueBuilder().
				WithNumber(123).
				WithTitle("Test Issue").
				Build(),
			phase: "Plan",
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager) {
				// セッション存在確認（あり）
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()

				// Window存在確認（なし）
				tmux.On("WindowExists", "test-session", "issue-123").Return(false, nil).Once()
				// Window作成（新規ウィンドウ判定付き）
				tmux.On("CreateWindowForIssueWithNewWindowDetection", "test-session", 123).
					Return("issue-123", true, nil).Once()

				// Worktree存在確認（なし）
				git.On("WorktreeExistsForIssue", mock.Anything, 123).Return(false, nil).Once()
				// Worktree作成
				git.On("CreateWorktreeForIssue", mock.Anything, 123).Return(nil).Once()

				// 新規ウィンドウの場合、pane検索を試みるがSetPaneTitleのみを実行
				// Pane検索（なし）
				tmux.On("GetPaneByTitle", "test-session", "issue-123", "Plan").
					Return(nil, assert.AnError).Once()
				// pane-base-index取得
				tmux.On("GetPaneBaseIndex").Return(0, nil).Once()
				// Pane 0のタイトル設定のみ
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
				// セッション存在確認（あり）
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()

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
				// セッション存在確認（あり）
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()

				// Window存在確認（あり）
				tmux.On("WindowExists", "test-session", "issue-789").Return(true, nil).Once()
				// 既存ウィンドウなのでCreateWindowForIssueWithNewWindowDetectionは呼ばれない

				// Worktree存在確認（あり）
				git.On("WorktreeExistsForIssue", mock.Anything, 789).Return(true, nil).Once()

				// 既存ウィンドウの場合、pane検索と分割を行う
				// Pane検索（なし）
				tmux.On("GetPaneByTitle", "test-session", "issue-789", "Implementation").
					Return(nil, assert.AnError).Once()
				// 新規pane作成（縦分割）
				tmux.On("CreatePane", "test-session", "issue-789", tmuxpkg.PaneOptions{
					Split:      "-h", // 縦分割
					Percentage: 50,
					Title:      "Implementation",
				}).Return(&tmuxpkg.PaneInfo{Index: 1, Title: "Implementation", Active: true}, nil).Once()

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
			name: "新規ウィンドウでのImplementationフェーズ - pane分割なし",
			issue: builders.NewIssueBuilder().
				WithNumber(888).
				WithTitle("Test Issue 4").
				Build(),
			phase: "Implementation",
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager) {
				// セッション存在確認（あり）
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()

				// Window存在確認（なし）
				tmux.On("WindowExists", "test-session", "issue-888").Return(false, nil).Once()
				// Window作成（新規ウィンドウ判定付き）
				tmux.On("CreateWindowForIssueWithNewWindowDetection", "test-session", 888).
					Return("issue-888", true, nil).Once()

				// Worktree存在確認（なし）
				git.On("WorktreeExistsForIssue", mock.Anything, 888).Return(false, nil).Once()
				// Worktree作成
				git.On("CreateWorktreeForIssue", mock.Anything, 888).Return(nil).Once()

				// 新規ウィンドウの場合、Implementationフェーズでもpane分割しない
				// Pane検索（なし）
				tmux.On("GetPaneByTitle", "test-session", "issue-888", "Implementation").
					Return(nil, assert.AnError).Once()
				// pane-base-index取得
				tmux.On("GetPaneBaseIndex").Return(0, nil).Once()
				// Pane 0のタイトル設定のみ
				tmux.On("SetPaneTitle", "test-session", "issue-888", 0, "Implementation").Return(nil).Once()

				// Worktreeパス取得
				git.On("GetWorktreePathForIssue", 888).Return("/test/worktree/issue-888").Once()
			},
			want: &WorkspaceInfo{
				WindowName:   "issue-888",
				WorktreePath: "/test/worktree/issue-888",
				PaneIndex:    0,
				PaneTitle:    "Implementation",
			},
			wantErr: false,
		},
		{
			name: "セッション存在確認失敗",
			issue: builders.NewIssueBuilder().
				WithNumber(222).
				WithTitle("Test Session Check Error").
				Build(),
			phase: "Plan",
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager) {
				// セッション存在確認（エラー）
				tmux.On("SessionExists", "test-session").Return(false, assert.AnError).Once()
			},
			want:        nil,
			wantErr:     true,
			errContains: "failed to check session existence",
		},
		{
			name: "セッション作成失敗",
			issue: builders.NewIssueBuilder().
				WithNumber(333).
				WithTitle("Test Session Create Error").
				Build(),
			phase: "Plan",
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager) {
				// セッション存在確認（なし）
				tmux.On("SessionExists", "test-session").Return(false, nil).Once()
				// セッション作成（エラー）
				tmux.On("EnsureSession", "test-session").Return(assert.AnError).Once()
			},
			want:        nil,
			wantErr:     true,
			errContains: "failed to ensure session",
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
				// セッション存在確認（あり）
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()

				// Window存在確認（なし）
				tmux.On("WindowExists", "test-session", "issue-999").Return(false, nil).Once()
				// Window作成失敗
				tmux.On("CreateWindowForIssueWithNewWindowDetection", "test-session", 999).
					Return("", false, assert.AnError).Once()
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

			// モックの設定
			tt.setupMocks(tmuxManager, worktreeManager)

			// BaseExecutorの作成
			executor := NewBaseExecutor(
				"test-session",
				tmuxManager,
				worktreeManager,
				nil, // ClaudeCommandBuilderは不要
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

// ExecuteInWorkspaceメソッドが削除されたため、このテストも削除
