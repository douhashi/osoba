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

// PlanAction はpane管理方式を使用する計画フェーズのアクション実装
type PlanAction struct {
	types.BaseAction
	baseExecutor *BaseExecutor
	sessionName  string
	stateManager StateManagerV2
	claudeConfig *claude.ClaudeConfig
	logger       logger.Logger
}

// NewPlanAction は新しいPlanActionを作成する
func NewPlanAction(
	sessionName string,
	tmuxManager tmuxpkg.Manager,
	stateManager StateManagerV2,
	worktreeManager git.WorktreeManager,
	claudeExecutor ClaudeCommandBuilder,
	claudeConfig *claude.ClaudeConfig,
	logger logger.Logger,
) *PlanAction {
	baseExecutor := NewBaseExecutor(
		sessionName,
		tmuxManager,
		worktreeManager,
		claudeExecutor,
		logger,
	)

	return &PlanAction{
		BaseAction:   types.BaseAction{Type: types.ActionTypePlan},
		baseExecutor: baseExecutor,
		sessionName:  sessionName,
		stateManager: stateManager,
		claudeConfig: claudeConfig,
		logger:       logger,
	}
}

// Execute は計画フェーズのアクションを実行する
func (a *PlanAction) Execute(ctx context.Context, issue *github.Issue) error {
	if issue == nil || issue.Number == nil {
		return fmt.Errorf("invalid issue")
	}

	issueNumber := int64(*issue.Number)
	a.logger.Info("Executing plan action", "issue_number", issueNumber)

	// 既に処理済みかチェック
	if a.stateManager.HasBeenProcessed(issueNumber, types.IssueStatePlan) {
		a.logger.Info("Issue has already been processed for plan phase", "issue_number", issueNumber)
		return nil
	}

	// 処理中かチェック
	if a.stateManager.IsProcessing(issueNumber) {
		return fmt.Errorf("issue #%d is already processing", issueNumber)
	}

	// 処理開始
	a.stateManager.SetState(issueNumber, types.IssueStatePlan, types.IssueStatusProcessing)

	// ワークスペースの準備
	workspace, err := a.baseExecutor.PrepareWorkspace(ctx, issue, "Plan")
	if err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStatePlan)
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
	phaseConfig, exists := a.claudeConfig.GetPhase("plan")
	if !exists {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStatePlan)
		return fmt.Errorf("plan phase config not found")
	}

	// Claudeコマンドの実行
	promptPath := phaseConfig.Prompt
	outputPath := fmt.Sprintf("tmp/execution_plan_%d.md", issueNumber)

	claudeCmd := a.baseExecutor.claudeExecutor.BuildCommand(
		promptPath,
		outputPath,
		workspace.WorktreePath,
		templateVars,
	)

	a.logger.Info("Executing Claude command",
		"issue_number", issueNumber,
		"command", claudeCmd,
	)

	// ワークスペースでClaudeコマンドを実行
	if err := a.baseExecutor.ExecuteInWorkspace(workspace, claudeCmd); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStatePlan)
		return fmt.Errorf("failed to execute Claude command: %w", err)
	}

	// 完了処理
	a.stateManager.MarkAsCompleted(issueNumber, types.IssueStatePlan)
	a.logger.Info("Plan action completed successfully", "issue_number", issueNumber)

	// V2ではフェーズ遷移は行わない（別のコンポーネントが管理）

	return nil
}

// CanExecute は計画フェーズのアクションが実行可能かを判定する
func (a *PlanAction) CanExecute(issue *github.Issue) bool {
	return hasLabel(issue, "status:needs-plan")
}

// worktreeConfig はworktreePath情報を保持する構造体
type worktreeConfig struct {
	WorktreePath string
}

// GetWorkDir はworktreeパスを返す
func (w worktreeConfig) GetWorkDir() string {
	return w.WorktreePath
}
