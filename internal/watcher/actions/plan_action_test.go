package actions

import (
	"context"
	"testing"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/testutil/builders"
	"github.com/douhashi/osoba/internal/testutil/helpers"
	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zapcore"
)

func TestPlanActionV2_Execute(t *testing.T) {
	tests := []struct {
		name         string
		issue        *github.Issue
		setupMocks   func(*mocks.MockTmuxManager, *mocks.MockGitWorktreeManager, *mocks.MockClaudeExecutor)
		claudeConfig *claude.ClaudeConfig
		wantErr      bool
		errContains  string
	}{
		{
			name: "正常なPlanアクション実行（argsあり）",
			issue: builders.NewIssueBuilder().
				WithNumber(123).
				WithTitle("Test Issue").
				WithLabel("status:needs-plan").
				Build(),
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager, claudeExec *mocks.MockClaudeExecutor) {
				// PrepareWorkspace
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()
				tmux.On("WindowExists", "test-session", "issue-123").Return(false, nil).Once()
				tmux.On("CreateWindowForIssueWithNewWindowDetection", "test-session", 123).
					Return("issue-123", true, nil).Once()
				git.On("WorktreeExistsForIssue", mock.Anything, 123).Return(false, nil).Once()
				git.On("CreateWorktreeForIssue", mock.Anything, 123).Return(nil).Once()
				tmux.On("GetPaneByTitle", "test-session", "issue-123", "Plan").Return(nil, assert.AnError).Once()
				tmux.On("GetPaneBaseIndex").Return(0, nil).Once()
				tmux.On("SetPaneTitle", "test-session", "issue-123", 0, "Plan").Return(nil).Once()
				tmux.On("ResizePanesEvenly", "test-session", "issue-123").Return(nil).Once() // 自動リサイズ機能
				git.On("GetWorktreePathForIssue", 123).Return("/test/worktree/issue-123").Once()

				// Claude実行 - ExecuteInTmuxを使用
				expectedConfig := &claude.PhaseConfig{
					Prompt: "prompts/plan.md",
					Args:   []string{"--arg1", "--arg2"},
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
			},
			claudeConfig: &claude.ClaudeConfig{
				Phases: map[string]*claude.PhaseConfig{
					"plan": {
						Prompt: "prompts/plan.md",
						Args:   []string{"--arg1", "--arg2"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "正常なPlanアクション実行（args空配列）",
			issue: builders.NewIssueBuilder().
				WithNumber(456).
				WithTitle("Test Issue 2").
				WithLabel("status:needs-plan").
				Build(),
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager, claudeExec *mocks.MockClaudeExecutor) {
				// PrepareWorkspace
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()
				tmux.On("WindowExists", "test-session", "issue-456").Return(true, nil).Once()
				git.On("WorktreeExistsForIssue", mock.Anything, 456).Return(true, nil).Once()
				tmux.On("GetPaneByTitle", "test-session", "issue-456", "Plan").Return(nil, assert.AnError).Once()
				tmux.On("GetPaneBaseIndex").Return(0, nil).Once()
				tmux.On("SetPaneTitle", "test-session", "issue-456", 0, "Plan").Return(nil).Once()
				tmux.On("ResizePanesEvenly", "test-session", "issue-456").Return(nil).Once() // 自動リサイズ機能
				git.On("GetWorktreePathForIssue", 456).Return("/test/worktree/issue-456").Once()

				// Claude実行 - ExecuteInTmuxを使用（args空配列）
				expectedConfig := &claude.PhaseConfig{
					Prompt: "prompts/plan.md",
					Args:   []string{},
				}
				claudeExec.On("ExecuteInTmux",
					mock.Anything,
					expectedConfig,
					mock.Anything,
					"test-session",
					"issue-456",
					"/test/worktree/issue-456",
				).Return(nil).Once()
			},
			claudeConfig: &claude.ClaudeConfig{
				Phases: map[string]*claude.PhaseConfig{
					"plan": {
						Prompt: "prompts/plan.md",
						Args:   []string{},
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "nilのissue",
			issue: nil,
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager, claudeExec *mocks.MockClaudeExecutor) {
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
				WithLabel("status:needs-plan").
				Build(),
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager, claudeExec *mocks.MockClaudeExecutor) {
				// PrepareWorkspace
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()
				tmux.On("WindowExists", "test-session", "issue-999").Return(true, nil).Once()
				git.On("WorktreeExistsForIssue", mock.Anything, 999).Return(true, nil).Once()
				tmux.On("GetPaneByTitle", "test-session", "issue-999", "Plan").Return(nil, assert.AnError).Once()
				tmux.On("GetPaneBaseIndex").Return(0, nil).Once()
				tmux.On("SetPaneTitle", "test-session", "issue-999", 0, "Plan").Return(nil).Once()
				tmux.On("ResizePanesEvenly", "test-session", "issue-999").Return(nil).Once() // 自動リサイズ機能
				git.On("GetWorktreePathForIssue", 999).Return("/test/worktree/issue-999").Once()
			},
			claudeConfig: &claude.ClaudeConfig{
				Phases: map[string]*claude.PhaseConfig{
					// planフェーズなし
				},
			},
			wantErr:     true,
			errContains: "plan phase config not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックの作成
			logger, _ := helpers.NewObservableLogger(zapcore.InfoLevel)
			tmuxManager := mocks.NewMockTmuxManager()
			worktreeManager := mocks.NewMockGitWorktreeManager()
			claudeExecutor := mocks.NewMockClaudeExecutor()

			// モックの設定
			tt.setupMocks(tmuxManager, worktreeManager, claudeExecutor)

			// アクションの作成
			cfg := config.NewConfig()
			action := NewPlanAction(
				"test-session",
				tmuxManager,
				worktreeManager,
				cfg,
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
		})
	}
}

func TestPlanActionV2_CanExecute(t *testing.T) {
	tests := []struct {
		name  string
		issue *github.Issue
		want  bool
	}{
		{
			name: "status:needs-planラベルあり",
			issue: builders.NewIssueBuilder().
				WithNumber(123).
				WithLabel("status:needs-plan").
				Build(),
			want: true,
		},
		{
			name: "status:needs-planラベルなし",
			issue: builders.NewIssueBuilder().
				WithNumber(456).
				WithLabel("status:ready").
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
			action := &PlanAction{
				logger: logger,
			}

			got := action.CanExecute(tt.issue)
			assert.Equal(t, tt.want, got)
		})
	}
}
