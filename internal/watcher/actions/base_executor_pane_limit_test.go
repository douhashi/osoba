package actions

import (
	"context"
	"testing"

	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/douhashi/osoba/internal/testutil/builders"
	"github.com/douhashi/osoba/internal/testutil/mocks"
	tmuxpkg "github.com/douhashi/osoba/internal/tmux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestBaseExecutor_EnsurePane_WithPaneLimit(t *testing.T) {
	tests := []struct {
		name       string
		phase      string
		config     *config.Config
		setupMocks func(*mocks.MockTmuxManager, *mocks.MockGitWorktreeManager)
		wantErr    bool
	}{
		{
			name:  "ペイン数制限有効 - 上限に達した場合非アクティブペインを削除",
			phase: "Review",
			config: &config.Config{
				Tmux: config.TmuxConfig{
					LimitPanesEnabled: true,
					MaxPanesPerWindow: 3,
				},
			},
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager) {
				// セッション存在確認
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()
				// Window存在確認（あり）
				tmux.On("WindowExists", "test-session", "issue-999").Return(true, nil).Once()
				// Worktree存在確認（あり）
				git.On("WorktreeExistsForIssue", mock.Anything, 999).Return(true, nil).Once()

				// 既存pane検索（なし）
				tmux.On("GetPaneByTitle", "test-session", "issue-999", "Review").
					Return(nil, assert.AnError).Once()

				// 新規pane作成（ペイン数制限機能が統合されたCreatePaneが呼ばれる）
				tmux.On("CreatePane", "test-session", "issue-999", tmuxpkg.PaneOptions{
					Split:      "-h",
					Percentage: 50,
					Title:      "Review",
					Config: &tmuxpkg.PaneConfig{
						LimitPanesEnabled: true,
						MaxPanesPerWindow: 3,
					},
				}).Return(&tmuxpkg.PaneInfo{Index: 3, Title: "Review", Active: true}, nil).Once()

				// Worktreeパス取得
				git.On("GetWorktreePathForIssue", 999).Return("/test/worktree/issue-999").Once()
			},
			wantErr: false,
		},
		{
			name:  "ペイン数制限有効 - 全ペインがアクティブの場合削除をスキップ",
			phase: "Review",
			config: &config.Config{
				Tmux: config.TmuxConfig{
					LimitPanesEnabled: true,
					MaxPanesPerWindow: 2,
				},
			},
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager) {
				// セッション存在確認
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()
				// Window存在確認（あり）
				tmux.On("WindowExists", "test-session", "issue-999").Return(true, nil).Once()
				// Worktree存在確認（あり）
				git.On("WorktreeExistsForIssue", mock.Anything, 999).Return(true, nil).Once()

				// 既存pane検索（なし）
				tmux.On("GetPaneByTitle", "test-session", "issue-999", "Review").
					Return(nil, assert.AnError).Once()

				// 新規pane作成（ペイン数制限機能が統合されたCreatePaneが呼ばれる）
				tmux.On("CreatePane", "test-session", "issue-999", tmuxpkg.PaneOptions{
					Split:      "-h",
					Percentage: 50,
					Title:      "Review",
					Config: &tmuxpkg.PaneConfig{
						LimitPanesEnabled: true,
						MaxPanesPerWindow: 2,
					},
				}).Return(&tmuxpkg.PaneInfo{Index: 2, Title: "Review", Active: true}, nil).Once()

				// Worktreeパス取得
				git.On("GetWorktreePathForIssue", 999).Return("/test/worktree/issue-999").Once()
			},
			wantErr: false,
		},
		{
			name:  "ペイン数制限無効 - 削除処理が実行されない",
			phase: "Review",
			config: &config.Config{
				Tmux: config.TmuxConfig{
					LimitPanesEnabled: false,
					MaxPanesPerWindow: 2,
				},
			},
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager) {
				// セッション存在確認
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()
				// Window存在確認（あり）
				tmux.On("WindowExists", "test-session", "issue-999").Return(true, nil).Once()
				// Worktree存在確認（あり）
				git.On("WorktreeExistsForIssue", mock.Anything, 999).Return(true, nil).Once()

				// 既存pane検索（なし）
				tmux.On("GetPaneByTitle", "test-session", "issue-999", "Review").
					Return(nil, assert.AnError).Once()

				// ListPanesは呼ばれない（制限無効のため）

				// 新規pane作成（制限無効なのでConfig=nil）
				tmux.On("CreatePane", "test-session", "issue-999", tmuxpkg.PaneOptions{
					Split:      "-h",
					Percentage: 50,
					Title:      "Review",
					Config:     nil,
				}).Return(&tmuxpkg.PaneInfo{Index: 3, Title: "Review", Active: true}, nil).Once()

				// Worktreeパス取得
				git.On("GetWorktreePathForIssue", 999).Return("/test/worktree/issue-999").Once()
			},
			wantErr: false,
		},
		{
			name:  "ペイン数制限有効 - 上限未満の場合削除しない",
			phase: "Implementation",
			config: &config.Config{
				Tmux: config.TmuxConfig{
					LimitPanesEnabled: true,
					MaxPanesPerWindow: 5,
				},
			},
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager) {
				// セッション存在確認
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()
				// Window存在確認（あり）
				tmux.On("WindowExists", "test-session", "issue-999").Return(true, nil).Once()
				// Worktree存在確認（あり）
				git.On("WorktreeExistsForIssue", mock.Anything, 999).Return(true, nil).Once()

				// 既存pane検索（なし）
				tmux.On("GetPaneByTitle", "test-session", "issue-999", "Implementation").
					Return(nil, assert.AnError).Once()

				// 新規pane作成（ペイン数制限機能が統合されたCreatePaneが呼ばれる）
				tmux.On("CreatePane", "test-session", "issue-999", tmuxpkg.PaneOptions{
					Split:      "-h",
					Percentage: 50,
					Title:      "Implementation",
					Config: &tmuxpkg.PaneConfig{
						LimitPanesEnabled: true,
						MaxPanesPerWindow: 5,
					},
				}).Return(&tmuxpkg.PaneInfo{Index: 2, Title: "Implementation", Active: true}, nil).Once()

				// Worktreeパス取得
				git.On("GetWorktreePathForIssue", 999).Return("/test/worktree/issue-999").Once()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックの準備
			mockTmux := mocks.NewMockTmuxManager()
			mockGit := mocks.NewMockGitWorktreeManager()
			logger, _ := logger.New(logger.WithLevel("debug"))

			// テストケースごとのモック設定
			tt.setupMocks(mockTmux, mockGit)

			// BaseExecutorの作成
			executor := NewBaseExecutor(
				"test-session",
				mockTmux,
				mockGit,
				tt.config,
				logger,
			)

			// テスト対象のissue
			issue := builders.NewIssueBuilder().
				WithNumber(999).
				WithTitle("Test Pane Limit").
				Build()

			// PrepareWorkspaceを実行
			ctx := context.Background()
			workspace, err := executor.PrepareWorkspace(ctx, issue, tt.phase)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, workspace)
				assert.Equal(t, "issue-999", workspace.WindowName)
				assert.Equal(t, tt.phase, workspace.PaneTitle)
			}

			// モックの期待値を確認
			mockTmux.AssertExpectations(t)
			mockGit.AssertExpectations(t)
		})
	}
}
