package actions

import (
	"context"
	"testing"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/testutil/builders"
	"github.com/douhashi/osoba/internal/testutil/helpers"
	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/douhashi/osoba/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zapcore"
)

func TestPlanActionV2_Execute(t *testing.T) {
	tests := []struct {
		name         string
		issue        *github.Issue
		setupMocks   func(*mocks.MockTmuxManager, *mocks.MockGitWorktreeManager, *mocks.MockClaudeCommandBuilder, *mocks.MockStateManager)
		claudeConfig *claude.ClaudeConfig
		wantErr      bool
		errContains  string
	}{
		{
			name: "正常なPlanアクション実行",
			issue: builders.NewIssueBuilder().
				WithNumber(123).
				WithTitle("Test Issue").
				WithLabel("status:needs-plan").
				Build(),
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager, claude *mocks.MockClaudeCommandBuilder, state *mocks.MockStateManager) {
				// 状態チェック
				state.On("HasBeenProcessed", int64(123), types.IssueStatePlan).Return(false).Once()
				state.On("IsProcessing", int64(123)).Return(false).Once()
				state.On("SetState", int64(123), types.IssueStatePlan, types.IssueStatusProcessing).Once()

				// PrepareWorkspace
				tmux.On("WindowExists", "test-session", "issue-123").Return(false, nil).Once()
				tmux.On("CreateWindowForIssueWithNewWindowDetection", "test-session", 123).
					Return("issue-123", true, nil).Once()
				git.On("WorktreeExistsForIssue", mock.Anything, 123).Return(false, nil).Once()
				git.On("CreateWorktreeForIssue", mock.Anything, 123).Return(nil).Once()
				tmux.On("GetPaneByTitle", "test-session", "issue-123", "Plan").Return(nil, assert.AnError).Once()
				tmux.On("SetPaneTitle", "test-session", "issue-123", 0, "Plan").Return(nil).Once()
				git.On("GetWorktreePathForIssue", 123).Return("/test/worktree/issue-123").Once()

				// Claude実行
				claude.On("BuildCommand",
					"prompts/plan.md",
					"tmp/execution_plan_123.md",
					"/test/worktree/issue-123",
					mock.Anything,
				).Return("claude plan command").Once()

				// RunInWindowを使用することを期待
				tmux.On("RunInWindow", "test-session", "issue-123", "cd /test/worktree/issue-123 && claude plan command").Return(nil).Once()

				// 完了処理
				state.On("MarkAsCompleted", int64(123), types.IssueStatePlan).Once()
			},
			claudeConfig: &claude.ClaudeConfig{
				Phases: map[string]*claude.PhaseConfig{
					"plan": {
						Prompt: "prompts/plan.md",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "既に処理済みのIssue",
			issue: builders.NewIssueBuilder().
				WithNumber(456).
				WithTitle("Already Processed").
				WithLabel("status:needs-plan").
				Build(),
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager, claude *mocks.MockClaudeCommandBuilder, state *mocks.MockStateManager) {
				state.On("HasBeenProcessed", int64(456), types.IssueStatePlan).Return(true).Once()
			},
			claudeConfig: &claude.ClaudeConfig{
				Phases: map[string]*claude.PhaseConfig{
					"plan": {
						Prompt: "prompts/plan.md",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "処理中のIssue",
			issue: builders.NewIssueBuilder().
				WithNumber(789).
				WithTitle("Processing").
				WithLabel("status:needs-plan").
				Build(),
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager, claude *mocks.MockClaudeCommandBuilder, state *mocks.MockStateManager) {
				state.On("HasBeenProcessed", int64(789), types.IssueStatePlan).Return(false).Once()
				state.On("IsProcessing", int64(789)).Return(true).Once()
			},
			claudeConfig: &claude.ClaudeConfig{
				Phases: map[string]*claude.PhaseConfig{
					"plan": {
						Prompt: "prompts/plan.md",
					},
				},
			},
			wantErr:     true,
			errContains: "is already processing",
		},
		{
			name:  "nilのissue",
			issue: nil,
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager, claude *mocks.MockClaudeCommandBuilder, state *mocks.MockStateManager) {
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
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager, claude *mocks.MockClaudeCommandBuilder, state *mocks.MockStateManager) {
				state.On("HasBeenProcessed", int64(999), types.IssueStatePlan).Return(false).Once()
				state.On("IsProcessing", int64(999)).Return(false).Once()
				state.On("SetState", int64(999), types.IssueStatePlan, types.IssueStatusProcessing).Once()

				// PrepareWorkspace
				tmux.On("WindowExists", "test-session", "issue-999").Return(true, nil).Once()
				git.On("WorktreeExistsForIssue", mock.Anything, 999).Return(true, nil).Once()
				tmux.On("GetPaneByTitle", "test-session", "issue-999", "Plan").Return(nil, assert.AnError).Once()
				tmux.On("SetPaneTitle", "test-session", "issue-999", 0, "Plan").Return(nil).Once()
				git.On("GetWorktreePathForIssue", 999).Return("/test/worktree/issue-999").Once()

				state.On("MarkAsFailed", int64(999), types.IssueStatePlan).Once()
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
			claudeExecutor := mocks.NewMockClaudeCommandBuilder()
			stateManager := mocks.NewMockStateManager()

			// モックの設定
			tt.setupMocks(tmuxManager, worktreeManager, claudeExecutor, stateManager)

			// アクションの作成
			action := NewPlanAction(
				"test-session",
				tmuxManager,
				stateManager,
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
			stateManager.AssertExpectations(t)
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
