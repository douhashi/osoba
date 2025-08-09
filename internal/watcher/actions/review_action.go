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

// ReviewAction はpane管理方式を使用するレビューフェーズのアクション実装
type ReviewAction struct {
	types.BaseAction
	baseExecutor   *BaseExecutor
	claudeExecutor claude.ClaudeExecutor
	sessionName    string
	labelManager   ActionsLabelManager
	claudeConfig   *claude.ClaudeConfig
	logger         logger.Logger
}

// NewReviewAction は新しいReviewActionを作成する
func NewReviewAction(
	sessionName string,
	tmuxManager tmuxpkg.Manager,
	labelManager ActionsLabelManager,
	worktreeManager git.WorktreeManager,
	claudeExecutor claude.ClaudeExecutor,
	claudeConfig *claude.ClaudeConfig,
	logger logger.Logger,
) *ReviewAction {
	baseExecutor := NewBaseExecutor(
		sessionName,
		tmuxManager,
		worktreeManager,
		nil, // ClaudeCommandBuilderは不要になったのでnilを渡す
		logger,
	)

	return &ReviewAction{
		BaseAction:     types.BaseAction{Type: types.ActionTypeReview},
		baseExecutor:   baseExecutor,
		claudeExecutor: claudeExecutor,
		sessionName:    sessionName,
		labelManager:   labelManager,
		claudeConfig:   claudeConfig,
		logger:         logger,
	}
}

// Execute はレビューフェーズのアクションを実行する
func (a *ReviewAction) Execute(ctx context.Context, issue *github.Issue) error {
	if issue == nil || issue.Number == nil {
		return fmt.Errorf("invalid issue")
	}

	issueNumber := int64(*issue.Number)
	a.logger.Info("Executing review action", "issue_number", issueNumber)

	// ワークスペースの準備
	workspace, err := a.baseExecutor.PrepareWorkspace(ctx, issue, "Review")
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
	phaseConfig, exists := a.claudeConfig.GetPhase("review")
	if !exists {
		return fmt.Errorf("review phase config not found")
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

	// ラベル更新: status:review-requested -> status:reviewed
	if a.labelManager != nil {
		a.logger.Info("Updating issue labels", "issue_number", issueNumber)
		if err := a.labelManager.RemoveLabel(ctx, int(issueNumber), "status:review-requested"); err != nil {
			a.logger.Error("Failed to remove label",
				"issue_number", issueNumber,
				"label", "status:review-requested",
				"error", err,
			)
		}
		if err := a.labelManager.AddLabel(ctx, int(issueNumber), "status:reviewed"); err != nil {
			a.logger.Error("Failed to add label",
				"issue_number", issueNumber,
				"label", "status:reviewed",
				"error", err,
			)
		}
	}

	a.logger.Info("Review action completed successfully", "issue_number", issueNumber)

	// V2ではフェーズ遷移は行わない（別のコンポーネントが管理）

	return nil
}

// CanExecute はレビューフェーズのアクションが実行可能かを判定する
func (a *ReviewAction) CanExecute(issue *github.Issue) bool {
	return hasLabel(issue, "status:review-requested")
}
