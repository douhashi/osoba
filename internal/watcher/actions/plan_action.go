package actions

import (
	"context"
	"fmt"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
	tmuxpkg "github.com/douhashi/osoba/internal/tmux"
	"github.com/douhashi/osoba/internal/types"
)

// PlanAction はpane管理方式を使用する計画フェーズのアクション実装
type PlanAction struct {
	types.BaseAction
	baseExecutor   *BaseExecutor
	claudeExecutor claude.ClaudeExecutor
	sessionName    string
	claudeConfig   *claude.ClaudeConfig
	logger         logger.Logger
}

// NewPlanAction は新しいPlanActionを作成する
func NewPlanAction(
	sessionName string,
	tmuxManager tmuxpkg.Manager,
	worktreeManager git.WorktreeManager,
	cfg *config.Config,
	claudeExecutor claude.ClaudeExecutor,
	claudeConfig *claude.ClaudeConfig,
	logger logger.Logger,
) *PlanAction {
	baseExecutor := NewBaseExecutor(
		sessionName,
		tmuxManager,
		worktreeManager,
		cfg,
		logger,
	)

	return &PlanAction{
		BaseAction:     types.BaseAction{Type: types.ActionTypePlan},
		baseExecutor:   baseExecutor,
		claudeExecutor: claudeExecutor,
		sessionName:    sessionName,
		claudeConfig:   claudeConfig,
		logger:         logger,
	}
}

// Execute は計画フェーズのアクションを実行する
func (a *PlanAction) Execute(ctx context.Context, issue *github.Issue) error {
	if issue == nil || issue.Number == nil {
		return fmt.Errorf("invalid issue")
	}

	issueNumber := int64(*issue.Number)
	a.logger.Info("[DEBUG] PlanAction.Execute called", "issue_number", issueNumber)
	a.logger.Info("Executing plan action", "issue_number", issueNumber)

	// ワークスペースの準備
	workspace, err := a.baseExecutor.PrepareWorkspace(ctx, issue, "Plan")
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
	phaseConfig, exists := a.claudeConfig.GetPhase("plan")
	if !exists {
		return fmt.Errorf("plan phase config not found")
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
