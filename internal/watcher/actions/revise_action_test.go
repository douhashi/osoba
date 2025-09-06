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

func TestReviseAction_Execute(t *testing.T) {
	tests := []struct {
		name         string
		issue        *github.Issue
		setupMocks   func(*mocks.MockTmuxManager, *mocks.MockGitWorktreeManager, *mocks.MockClaudeExecutor, *mocks.MockLabelManager)
		claudeConfig *claude.ClaudeConfig
		wantErr      bool
		errContains  string
	}{
		{
			name: "正常なReviseアクション実行（PR存在）",
			issue: builders.NewIssueBuilder().
				WithNumber(123).
				WithTitle("Test Issue").
				WithLabel("status:requires-changes").
				Build(),
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager, claudeExec *mocks.MockClaudeExecutor, labelManager *mocks.MockLabelManager) {
				// PRラベル削除のため、PRを取得
				pr := &github.PullRequest{
					Number: 456,
				}
				labelManager.On("GetPullRequestForIssue", mock.Anything, 123).Return(pr, nil).Once()
				labelManager.On("RemoveLabel", mock.Anything, 456, "status:requires-changes").Return(nil).Once()

				// PrepareWorkspace - 既存のワークスペースを再利用
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()
				tmux.On("WindowExists", "test-session", "issue-123").Return(true, nil).Once()
				git.On("WorktreeExistsForIssue", mock.Anything, 123).Return(true, nil).Once()
				tmux.On("GetPaneByTitle", "test-session", "issue-123", "Revise").Return(nil, assert.AnError).Once()
				// Reviseフェーズでは新しいpaneを作成
				tmux.On("CreatePane", "test-session", "issue-123", mock.Anything).
					Return(&tmuxpkg.PaneInfo{Index: 2, Title: "Revise", Active: true}, nil).Once()
				git.On("GetWorktreePathForIssue", 123).Return("/test/worktree/issue-123").Once()

				// Claude実行 - ExecuteInTmuxを使用
				expectedConfig := &claude.PhaseConfig{
					Prompt: "/osoba:revise {{issue-number}}",
					Args:   []string{"--dangerously-skip-permissions"},
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
				labelManager.On("RemoveLabel", mock.Anything, 123, "status:requires-changes").Return(nil).Once()
				labelManager.On("RemoveLabel", mock.Anything, 123, "status:reviewing").Return(nil).Once()
				labelManager.On("AddLabel", mock.Anything, 123, "status:revising").Return(nil).Once()
			},
			claudeConfig: &claude.ClaudeConfig{
				Phases: map[string]*claude.PhaseConfig{
					"revise": {
						Prompt: "/osoba:revise {{issue-number}}",
						Args:   []string{"--dangerously-skip-permissions"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "正常なReviseアクション実行（PR存在しない）",
			issue: builders.NewIssueBuilder().
				WithNumber(124).
				WithTitle("Test Issue No PR").
				WithLabel("status:requires-changes").
				Build(),
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager, claudeExec *mocks.MockClaudeExecutor, labelManager *mocks.MockLabelManager) {
				// PRラベル削除のため、PRを取得（PRが存在しない）
				labelManager.On("GetPullRequestForIssue", mock.Anything, 124).Return(nil, nil).Once()
				// PRが存在しない場合はPRラベル削除をスキップ

				// PrepareWorkspace - 既存のワークスペースを再利用
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()
				tmux.On("WindowExists", "test-session", "issue-124").Return(true, nil).Once()
				git.On("WorktreeExistsForIssue", mock.Anything, 124).Return(true, nil).Once()
				tmux.On("GetPaneByTitle", "test-session", "issue-124", "Revise").Return(nil, assert.AnError).Once()
				// Reviseフェーズでは新しいpaneを作成
				tmux.On("CreatePane", "test-session", "issue-124", mock.Anything).
					Return(&tmuxpkg.PaneInfo{Index: 2, Title: "Revise", Active: true}, nil).Once()
				git.On("GetWorktreePathForIssue", 124).Return("/test/worktree/issue-124").Once()

				// Claude実行 - ExecuteInTmuxを使用
				expectedConfig := &claude.PhaseConfig{
					Prompt: "/osoba:revise {{issue-number}}",
					Args:   []string{"--dangerously-skip-permissions"},
				}
				expectedVars := &claude.TemplateVariables{
					IssueNumber: 124,
					IssueTitle:  "Test Issue No PR",
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
					"issue-124",
					"/test/worktree/issue-124",
				).Return(nil).Once()

				// ラベル更新
				labelManager.On("RemoveLabel", mock.Anything, 124, "status:requires-changes").Return(nil).Once()
				labelManager.On("RemoveLabel", mock.Anything, 124, "status:reviewing").Return(nil).Once()
				labelManager.On("AddLabel", mock.Anything, 124, "status:revising").Return(nil).Once()
			},
			claudeConfig: &claude.ClaudeConfig{
				Phases: map[string]*claude.PhaseConfig{
					"revise": {
						Prompt: "/osoba:revise {{issue-number}}",
						Args:   []string{"--dangerously-skip-permissions"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "PRラベル削除失敗でも処理継続",
			issue: builders.NewIssueBuilder().
				WithNumber(125).
				WithTitle("Test Issue PR Label Error").
				WithLabel("status:requires-changes").
				Build(),
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager, claudeExec *mocks.MockClaudeExecutor, labelManager *mocks.MockLabelManager) {
				// PRラベル削除のため、PRを取得
				pr := &github.PullRequest{
					Number: 457,
				}
				labelManager.On("GetPullRequestForIssue", mock.Anything, 125).Return(pr, nil).Once()
				labelManager.On("RemoveLabel", mock.Anything, 457, "status:requires-changes").Return(assert.AnError).Once()
				// エラーでも処理継続

				// PrepareWorkspace - 既存のワークスペースを再利用
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()
				tmux.On("WindowExists", "test-session", "issue-125").Return(true, nil).Once()
				git.On("WorktreeExistsForIssue", mock.Anything, 125).Return(true, nil).Once()
				tmux.On("GetPaneByTitle", "test-session", "issue-125", "Revise").Return(nil, assert.AnError).Once()
				// Reviseフェーズでは新しいpaneを作成
				tmux.On("CreatePane", "test-session", "issue-125", mock.Anything).
					Return(&tmuxpkg.PaneInfo{Index: 2, Title: "Revise", Active: true}, nil).Once()
				git.On("GetWorktreePathForIssue", 125).Return("/test/worktree/issue-125").Once()

				// Claude実行 - ExecuteInTmuxを使用
				expectedConfig := &claude.PhaseConfig{
					Prompt: "/osoba:revise {{issue-number}}",
					Args:   []string{"--dangerously-skip-permissions"},
				}
				expectedVars := &claude.TemplateVariables{
					IssueNumber: 125,
					IssueTitle:  "Test Issue PR Label Error",
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
					"issue-125",
					"/test/worktree/issue-125",
				).Return(nil).Once()

				// ラベル更新
				labelManager.On("RemoveLabel", mock.Anything, 125, "status:requires-changes").Return(nil).Once()
				labelManager.On("RemoveLabel", mock.Anything, 125, "status:reviewing").Return(nil).Once()
				labelManager.On("AddLabel", mock.Anything, 125, "status:revising").Return(nil).Once()
			},
			claudeConfig: &claude.ClaudeConfig{
				Phases: map[string]*claude.PhaseConfig{
					"revise": {
						Prompt: "/osoba:revise {{issue-number}}",
						Args:   []string{"--dangerously-skip-permissions"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "status:reviewingラベル削除失敗でも処理継続",
			issue: builders.NewIssueBuilder().
				WithNumber(126).
				WithTitle("Test Issue Reviewing Label Error").
				WithLabel("status:requires-changes").
				Build(),
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager, claudeExec *mocks.MockClaudeExecutor, labelManager *mocks.MockLabelManager) {
				// PRラベル削除のため、PRを取得（PRが存在しない）
				labelManager.On("GetPullRequestForIssue", mock.Anything, 126).Return(nil, nil).Once()
				// PRが存在しない場合はPRラベル削除をスキップ

				// PrepareWorkspace - 既存のワークスペースを再利用
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()
				tmux.On("WindowExists", "test-session", "issue-126").Return(true, nil).Once()
				git.On("WorktreeExistsForIssue", mock.Anything, 126).Return(true, nil).Once()
				tmux.On("GetPaneByTitle", "test-session", "issue-126", "Revise").Return(nil, assert.AnError).Once()
				// Reviseフェーズでは新しいpaneを作成
				tmux.On("CreatePane", "test-session", "issue-126", mock.Anything).
					Return(&tmuxpkg.PaneInfo{Index: 2, Title: "Revise", Active: true}, nil).Once()
				git.On("GetWorktreePathForIssue", 126).Return("/test/worktree/issue-126").Once()

				// Claude実行 - ExecuteInTmuxを使用
				expectedConfig := &claude.PhaseConfig{
					Prompt: "/osoba:revise {{issue-number}}",
					Args:   []string{"--dangerously-skip-permissions"},
				}
				expectedVars := &claude.TemplateVariables{
					IssueNumber: 126,
					IssueTitle:  "Test Issue Reviewing Label Error",
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
					"issue-126",
					"/test/worktree/issue-126",
				).Return(nil).Once()

				// ラベル更新
				labelManager.On("RemoveLabel", mock.Anything, 126, "status:requires-changes").Return(nil).Once()
				// status:reviewingラベル削除でエラーが発生
				labelManager.On("RemoveLabel", mock.Anything, 126, "status:reviewing").Return(assert.AnError).Once()
				// エラーでも処理継続
				labelManager.On("AddLabel", mock.Anything, 126, "status:revising").Return(nil).Once()
			},
			claudeConfig: &claude.ClaudeConfig{
				Phases: map[string]*claude.PhaseConfig{
					"revise": {
						Prompt: "/osoba:revise {{issue-number}}",
						Args:   []string{"--dangerously-skip-permissions"},
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
			name: "issue numberがnil",
			issue: &github.Issue{
				Title: github.String("Test Issue"),
			},
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
				WithLabel("status:requires-changes").
				Build(),
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager, claudeExec *mocks.MockClaudeExecutor, labelManager *mocks.MockLabelManager) {
				// PRラベル削除処理
				labelManager.On("GetPullRequestForIssue", mock.Anything, 999).Return(nil, nil).Once()

				// PrepareWorkspace
				tmux.On("SessionExists", "test-session").Return(true, nil).Once()
				tmux.On("WindowExists", "test-session", "issue-999").Return(true, nil).Once()
				git.On("WorktreeExistsForIssue", mock.Anything, 999).Return(true, nil).Once()
				tmux.On("GetPaneByTitle", "test-session", "issue-999", "Revise").Return(nil, assert.AnError).Once()
				tmux.On("CreatePane", "test-session", "issue-999", mock.Anything).
					Return(&tmuxpkg.PaneInfo{Index: 1, Title: "Revise", Active: true}, nil).Once()
				git.On("GetWorktreePathForIssue", 999).Return("/test/worktree/issue-999").Once()
			},
			claudeConfig: &claude.ClaudeConfig{
				Phases: map[string]*claude.PhaseConfig{
					// reviseフェーズなし
				},
			},
			wantErr:     true,
			errContains: "revise phase config not found",
		},
		{
			name: "ワークスペース準備失敗",
			issue: builders.NewIssueBuilder().
				WithNumber(456).
				WithTitle("Workspace Error").
				WithLabel("status:requires-changes").
				Build(),
			setupMocks: func(tmux *mocks.MockTmuxManager, git *mocks.MockGitWorktreeManager, claudeExec *mocks.MockClaudeExecutor, labelManager *mocks.MockLabelManager) {
				// PRラベル削除処理
				labelManager.On("GetPullRequestForIssue", mock.Anything, 456).Return(nil, nil).Once()

				tmux.On("SessionExists", "test-session").Return(false, assert.AnError).Once()
			},
			claudeConfig: &claude.ClaudeConfig{
				Phases: map[string]*claude.PhaseConfig{
					"revise": {
						Prompt: "/osoba:revise {{issue-number}}",
						Args:   []string{"--dangerously-skip-permissions"},
					},
				},
			},
			wantErr:     true,
			errContains: "failed to prepare workspace",
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
			action := NewReviseAction(
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

			// モックの呼び出し確認
			tmuxManager.AssertExpectations(t)
			worktreeManager.AssertExpectations(t)
			claudeExecutor.AssertExpectations(t)
			labelManager.AssertExpectations(t)
		})
	}
}

func TestReviseAction_CanExecute(t *testing.T) {
	tests := []struct {
		name     string
		issue    *github.Issue
		expected bool
	}{
		{
			name: "status:requires-changesラベルを持つissue",
			issue: builders.NewIssueBuilder().
				WithNumber(123).
				WithLabel("status:requires-changes").
				Build(),
			expected: true,
		},
		{
			name: "status:requires-changesラベルを持たないissue",
			issue: builders.NewIssueBuilder().
				WithNumber(456).
				WithLabel("status:ready").
				Build(),
			expected: false,
		},
		{
			name: "ラベルなしのissue",
			issue: builders.NewIssueBuilder().
				WithNumber(789).
				Build(),
			expected: false,
		},
		{
			name:     "nilのissue",
			issue:    nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := helpers.NewObservableLogger(zapcore.InfoLevel)
			action := NewReviseAction(
				"test-session",
				mocks.NewMockTmuxManager(),
				mocks.NewMockLabelManager(),
				mocks.NewMockGitWorktreeManager(),
				mocks.NewMockClaudeExecutor(),
				&claude.ClaudeConfig{},
				logger,
			)

			result := action.CanExecute(tt.issue)
			assert.Equal(t, tt.expected, result)
		})
	}
}
