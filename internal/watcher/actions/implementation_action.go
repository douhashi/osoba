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

// ImplementationAction はpane管理方式を使用する実装フェーズのアクション実装
type ImplementationAction struct {
	types.BaseAction
	baseExecutor *BaseExecutor
	sessionName  string
	stateManager StateManagerV2
	labelManager ActionsLabelManager
	claudeConfig *claude.ClaudeConfig
	logger       logger.Logger
}

// NewImplementationAction は新しいImplementationActionを作成する
func NewImplementationAction(
	sessionName string,
	tmuxManager tmuxpkg.Manager,
	stateManager StateManagerV2,
	labelManager ActionsLabelManager,
	worktreeManager git.WorktreeManager,
	claudeExecutor ClaudeCommandBuilder,
	claudeConfig *claude.ClaudeConfig,
	logger logger.Logger,
) *ImplementationAction {
	baseExecutor := NewBaseExecutor(
		sessionName,
		tmuxManager,
		worktreeManager,
		claudeExecutor,
		logger,
	)

	return &ImplementationAction{
		BaseAction:   types.BaseAction{Type: types.ActionTypeImplementation},
		baseExecutor: baseExecutor,
		sessionName:  sessionName,
		stateManager: stateManager,
		labelManager: labelManager,
		claudeConfig: claudeConfig,
		logger:       logger,
	}
}

// Execute は実装フェーズのアクションを実行する
func (a *ImplementationAction) Execute(ctx context.Context, issue *github.Issue) error {
	if issue == nil || issue.Number == nil {
		return fmt.Errorf("invalid issue")
	}

	issueNumber := int64(*issue.Number)
	a.logger.Info("Executing implementation action", "issue_number", issueNumber)

	// 既に処理済みかチェック
	if a.stateManager.HasBeenProcessed(issueNumber, types.IssueStateImplementation) {
		a.logger.Info("Issue has already been processed for implementation phase", "issue_number", issueNumber)
		return nil
	}

	// 処理中かチェック
	if a.stateManager.IsProcessing(issueNumber) {
		return fmt.Errorf("issue #%d is already processing", issueNumber)
	}

	// 処理開始
	a.stateManager.SetState(issueNumber, types.IssueStateImplementation, types.IssueStatusProcessing)

	// ワークスペースの準備
	workspace, err := a.baseExecutor.PrepareWorkspace(ctx, issue, "Implementation")
	if err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateImplementation)
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
	phaseConfig, exists := a.claudeConfig.GetPhase("implement")
	if !exists {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateImplementation)
		return fmt.Errorf("implement phase config not found")
	}

	// Claudeコマンドの実行
	promptPath := phaseConfig.Prompt

	claudeCmd := a.baseExecutor.claudeExecutor.BuildCommand(
		promptPath,
		"", // 実装フェーズでは出力ファイルは使用しない
		workspace.WorktreePath,
		templateVars,
	)

	a.logger.Info("Executing Claude command",
		"issue_number", issueNumber,
		"command", claudeCmd,
	)

	// ワークスペースでClaudeコマンドを実行
	if err := a.baseExecutor.ExecuteInWorkspace(workspace, claudeCmd); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateImplementation)
		return fmt.Errorf("failed to execute Claude command: %w", err)
	}

	// ラベル更新: status:ready -> status:implementing
	if a.labelManager != nil {
		a.logger.Info("Updating issue labels", "issue_number", issueNumber)
		if err := a.labelManager.RemoveLabel(ctx, int(issueNumber), "status:ready"); err != nil {
			a.logger.Error("Failed to remove label",
				"issue_number", issueNumber,
				"label", "status:ready",
				"error", err,
			)
		}
		if err := a.labelManager.AddLabel(ctx, int(issueNumber), "status:implementing"); err != nil {
			a.logger.Error("Failed to add label",
				"issue_number", issueNumber,
				"label", "status:implementing",
				"error", err,
			)
		}
	}

	// 完了処理
	a.stateManager.MarkAsCompleted(issueNumber, types.IssueStateImplementation)
	a.logger.Info("Implementation action completed successfully", "issue_number", issueNumber)

	// V2ではフェーズ遷移は行わない（別のコンポーネントが管理）

	return nil
}

// CanExecute は実装フェーズのアクションが実行可能かを判定する
func (a *ImplementationAction) CanExecute(issue *github.Issue) bool {
	return hasLabel(issue, "status:ready")
}
