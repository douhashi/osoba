package actions

import (
	"context"
	"testing"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/testutil/builders"
	"github.com/douhashi/osoba/internal/testutil/helpers"
	"github.com/douhashi/osoba/internal/testutil/mocks"
	tmuxpkg "github.com/douhashi/osoba/internal/tmux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zapcore"
)

func TestReviewAction_Execute(t *testing.T) {
	tests := []struct {
		name         string
		issue        *github.Issue
		setupMocks   func(*mocks.MockTmuxManager, *mocks.MockGitWorktreeManager, *mocks.MockClaudeExecutor, *mocks.MockLabelManager)
		claudeConfig *claude.ClaudeConfig
		wantErr      bool
		errContains  string
	}{
		{
			name: "正常なReviewアクション実行",
			issue: builders.NewIssueBuilder().
				WithNumber(123).
				WithTitle("Test Issue").
				WithLabel("status:review-requested").
				Build(),
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager, claudeExec *mocks.MockClaudeExecutor, labelManager *mocks.MockLabelManager) {
				// PrepareWorkspace
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()
				tmux.On("WindowExists", "test-session", "issue-123").Return(false, nil).Once()
				tmux.On("CreateWindowForIssueWithNewWindowDetection", "test-session", 123).
					Return("issue-123", true, nil).Once()
				git.On("WorktreeExistsForIssue", mock.Anything, 123).Return(false, nil).Once()
				git.On("CreateWorktreeForIssue", mock.Anything, 123).Return(nil).Once()
				tmux.On("GetPaneByTitle", "test-session", "issue-123", "Review").Return(nil, assert.AnError).Once()
				// 新しいウィンドウの場合はGetPaneBaseIndexとSetPaneTitleが呼ばれる
				tmux.On("GetPaneBaseIndex").Return(0, nil).Once()
				tmux.On("SetPaneTitle", "test-session", "issue-123", 0, "Review").Return(nil).Once()
				git.On("GetWorktreePathForIssue", 123).Return("/test/worktree/issue-123").Once()

				// Claude実行 - ExecuteInTmuxを使用
				expectedConfig := &claude.PhaseConfig{
					Prompt: "prompts/review.md",
					Args:   []string{"--review"},
				}
				expectedVars := &claude.TemplateVariables{
					IssueNumber: 123,
					IssueTitle:  "Test Issue",
					RepoName:    "osoba",
				}
				claudeExec.On("ExecuteInTmux",
					mock.Anything,
					expectedConfig,
					mock.MatchedBy(func(vars *claude.TemplateVariables) bool {
						return vars.IssueNumber == expectedVars.IssueNumber &&
							vars.IssueTitle == expectedVars.IssueTitle
					}),
					"test-session",
					"issue-123",
					"/test/worktree/issue-123",
				).Return(nil).Once()

				// ラベル更新
				labelManager.On("RemoveLabel", mock.Anything, 123, "status:review-requested").Return(nil).Once()
				labelManager.On("AddLabel", mock.Anything, 123, "status:reviewed").Return(nil).Once()
			},
			claudeConfig: &claude.ClaudeConfig{
				Phases: map[string]*claude.PhaseConfig{
					"review": {
						Prompt: "prompts/review.md",
						Args:   []string{"--review"},
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "nilのissue",
			issue: nil,
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager, claudeExec *mocks.MockClaudeExecutor, labelManager *mocks.MockLabelManager) {
				// 何も呼ばれない
			},
			claudeConfig: &claude.ClaudeConfig{},
			wantErr:      true,
			errContains:  "invalid issue",
		},
		{
			name: "phase設定が見つからない",
			issue: builders.NewIssueBuilder().
				WithNumber(999).
				WithTitle("No Config").
				WithLabel("status:review-requested").
				Build(),
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager, claudeExec *mocks.MockClaudeExecutor, labelManager *mocks.MockLabelManager) {
				// PrepareWorkspace
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()
				tmux.On("WindowExists", "test-session", "issue-999").Return(true, nil).Once()
				git.On("WorktreeExistsForIssue", mock.Anything, 999).Return(true, nil).Once()
				tmux.On("GetPaneByTitle", "test-session", "issue-999", "Review").Return(nil, assert.AnError).Once()
				// Reviewフェーズでは新しいpaneを作成
				tmux.On("CreatePane", "test-session", "issue-999", mock.Anything).
					Return(&tmuxpkg.PaneInfo{Index: 1, Title: "Review", Active: true}, nil).Once()
				git.On("GetWorktreePathForIssue", 999).Return("/test/worktree/issue-999").Once()
			},
			claudeConfig: &claude.ClaudeConfig{
				Phases: map[string]*claude.PhaseConfig{
					// reviewフェーズなし
				},
			},
			wantErr:     true,
			errContains: "review phase config not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックの作成
			logger, _ := helpers.NewObservableLogger(zapcore.InfoLevel)
			tmuxManager := mocks.NewMockTmuxManager()
			worktreeManager := mocks.NewMockGitWorktreeManager()
			claudeExecutor := mocks.NewMockClaudeExecutor()
			labelManager := mocks.NewMockLabelManager()

			// モックの設定
			tt.setupMocks(tmuxManager, worktreeManager, claudeExecutor, labelManager)

			// アクションの作成
			action := NewReviewAction(
				"test-session",
				tmuxManager,
				labelManager,
				worktreeManager,
				claudeExecutor,
				tt.claudeConfig,
				logger,
			)

			// テスト実行
			err := action.Execute(context.Background(), tt.issue)

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
			worktreeManager.AssertExpectations(t)
			claudeExecutor.AssertExpectations(t)
			labelManager.AssertExpectations(t)
		})
	}
}

func TestReviewAction_CanExecute(t *testing.T) {
	tests := []struct {
		name  string
		issue *github.Issue
		want  bool
	}{
		{
			name: "status:review-requestedラベルあり",
			issue: builders.NewIssueBuilder().
				WithNumber(123).
				WithLabel("status:review-requested").
				Build(),
			want: true,
		},
		{
			name: "status:review-requestedラベルなし",
			issue: builders.NewIssueBuilder().
				WithNumber(456).
				WithLabel("status:implementing").
				Build(),
			want: false,
		},
		{
			name: "ラベルなし",
			issue: builders.NewIssueBuilder().
				WithNumber(789).
				Build(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := helpers.NewObservableLogger(zapcore.InfoLevel)
			action := &ReviewAction{
				logger: logger,
			}

			got := action.CanExecute(tt.issue)
			assert.Equal(t, tt.want, got)
		})
	}
}
