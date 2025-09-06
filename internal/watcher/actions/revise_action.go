package actions

import (
	"context"
	"fmt"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
	tmuxpkg "github.com/douhashi/osoba/internal/tmux"
	"github.com/douhashi/osoba/internal/types"
)

// ReviseAction はpane管理方式を使用するレビュー指摘対応フェーズのアクション実装
type ReviseAction struct {
	types.BaseAction
	baseExecutor   *BaseExecutor
	claudeExecutor claude.ClaudeExecutor
	sessionName    string
	labelManager   ActionsLabelManager
	claudeConfig   *claude.ClaudeConfig
	logger         logger.Logger
}

// NewReviseAction は新しいReviseActionを作成する
func NewReviseAction(
	sessionName string,
	tmuxManager tmuxpkg.Manager,
	labelManager ActionsLabelManager,
	worktreeManager git.WorktreeManager,
	claudeExecutor claude.ClaudeExecutor,
	claudeConfig *claude.ClaudeConfig,
	logger logger.Logger,
) *ReviseAction {
	baseExecutor := NewBaseExecutor(
		sessionName,
		tmuxManager,
		worktreeManager,
		nil, // ClaudeCommandBuilderは不要になったのでnilを渡す
		logger,
	)

	return &ReviseAction{
		BaseAction:     types.BaseAction{Type: types.ActionTypeRevise},
		baseExecutor:   baseExecutor,
		claudeExecutor: claudeExecutor,
		sessionName:    sessionName,
		labelManager:   labelManager,
		claudeConfig:   claudeConfig,
		logger:         logger,
	}
}

// Execute はレビュー指摘対応フェーズのアクションを実行する
func (a *ReviseAction) Execute(ctx context.Context, issue *github.Issue) error {
	if issue == nil || issue.Number == nil {
		return fmt.Errorf("invalid issue")
	}

	issueNumber := int64(*issue.Number)
	a.logger.Info("Executing revise action", "issue_number", issueNumber)

	// PRのstatus:requires-changesラベルを削除（重複実行防止）
	if a.labelManager != nil {
		pr, err := a.labelManager.GetPullRequestForIssue(ctx, int(issueNumber))
		if err != nil {
			a.logger.Error("Failed to get PR for issue",
				"issue_number", issueNumber,
				"error", err,
			)
			// PRが取得できない場合も処理を継続
		} else if pr != nil {
			a.logger.Info("Found PR for issue, removing PR label",
				"issue_number", issueNumber,
				"pr_number", pr.Number,
			)
			// PRのstatus:requires-changesラベルを削除
			if err := a.labelManager.RemoveLabel(ctx, pr.Number, "status:requires-changes"); err != nil {
				a.logger.Error("Failed to remove PR label",
					"pr_number", pr.Number,
					"label", "status:requires-changes",
					"error", err,
				)
				// エラーが発生しても処理を継続
			} else {
				a.logger.Info("Successfully removed PR label",
					"pr_number", pr.Number,
					"label", "status:requires-changes",
				)
			}
		} else {
			a.logger.Info("No PR found for issue",
				"issue_number", issueNumber,
			)
		}
	}

	// ワークスペースの準備（既存のものを再利用）
	workspace, err := a.baseExecutor.PrepareWorkspace(ctx, issue, "Revise")
	if err != nil {
		return fmt.Errorf("failed to prepare workspace: %w", err)
	}

	a.logger.Info("Workspace prepared",
		"issue_number", issueNumber,
		"window_name", workspace.WindowName,
		"worktree_path", workspace.WorktreePath,
		"pane_index", workspace.PaneIndex,
	)

	// Claude実行用の変数を準備
	templateVars := &claude.TemplateVariables{
		IssueNumber: int(issueNumber),
		IssueTitle:  getIssueTitle(issue),
		RepoName:    getRepoName(),
	}

	// Claude設定を取得
	phaseConfig, exists := a.claudeConfig.GetPhase("revise")
	if !exists {
		return fmt.Errorf("revise phase config not found")
	}

	// ClaudeExecutorを使用してtmuxウィンドウ内で実行
	a.logger.Info("Executing Claude in tmux window",
		"issue_number", issueNumber,
		"session", a.sessionName,
		"window", workspace.WindowName,
		"worktree_path", workspace.WorktreePath,
	)

	if err := a.claudeExecutor.ExecuteInTmux(ctx, phaseConfig, templateVars, a.sessionName, workspace.WindowName, workspace.WorktreePath); err != nil {
		return fmt.Errorf("failed to execute Claude command: %w", err)
	}

	// ラベル更新: status:requires-changes -> status:revising
	if a.labelManager != nil {
		a.logger.Info("Updating issue labels", "issue_number", issueNumber)
		if err := a.labelManager.RemoveLabel(ctx, int(issueNumber), "status:requires-changes"); err != nil {
			a.logger.Error("Failed to remove label",
				"issue_number", issueNumber,
				"label", "status:requires-changes",
				"error", err,
			)
		}
		// status:reviewingラベルも削除（存在しない場合やエラーでも処理継続）
		if err := a.labelManager.RemoveLabel(ctx, int(issueNumber), "status:reviewing"); err != nil {
			a.logger.Error("Failed to remove label",
				"issue_number", issueNumber,
				"label", "status:reviewing",
				"error", err,
			)
		}
		if err := a.labelManager.AddLabel(ctx, int(issueNumber), "status:revising"); err != nil {
			a.logger.Error("Failed to add label",
				"issue_number", issueNumber,
				"label", "status:revising",
				"error", err,
			)
		}
	}

	a.logger.Info("Revise action completed successfully", "issue_number", issueNumber)

	// V2ではフェーズ遷移は行わない（別のコンポーネントが管理）

	return nil
}

// CanExecute はレビュー指摘対応フェーズのアクションが実行可能かを判定する
func (a *ReviseAction) CanExecute(issue *github.Issue) bool {
	return hasLabel(issue, "status:requires-changes")
}
