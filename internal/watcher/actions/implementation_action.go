package actions

import (
	"context"
	"fmt"
	"log"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/types"
	"github.com/google/go-github/v67/github"
)

// ImplementationAction は実装フェーズのアクション実装
type ImplementationAction struct {
	types.BaseAction
	sessionName     string
	tmuxClient      TmuxClient
	stateManager    StateManager
	labelManager    LabelManager
	worktreeManager git.WorktreeManager
	claudeExecutor  claude.ClaudeExecutor
	claudeConfig    *claude.ClaudeConfig
}

// NewImplementationAction は新しいImplementationActionを作成する
func NewImplementationAction(
	sessionName string,
	tmuxClient TmuxClient,
	stateManager StateManager,
	labelManager LabelManager,
	worktreeManager git.WorktreeManager,
	claudeExecutor claude.ClaudeExecutor,
	claudeConfig *claude.ClaudeConfig,
) *ImplementationAction {
	return &ImplementationAction{
		BaseAction:      types.BaseAction{Type: types.ActionTypeImplementation},
		sessionName:     sessionName,
		tmuxClient:      tmuxClient,
		stateManager:    stateManager,
		labelManager:    labelManager,
		worktreeManager: worktreeManager,
		claudeExecutor:  claudeExecutor,
		claudeConfig:    claudeConfig,
	}
}

// Execute は実装フェーズのアクションを実行する
func (a *ImplementationAction) Execute(ctx context.Context, issue *github.Issue) error {
	if issue == nil || issue.Number == nil {
		return fmt.Errorf("invalid issue")
	}

	issueNumber := int64(*issue.Number)
	log.Printf("Executing implementation action for issue #%d", issueNumber)

	// 既に処理済みかチェック
	if a.stateManager.HasBeenProcessed(issueNumber, types.IssueStateImplementation) {
		log.Printf("Issue #%d has already been processed for implementation phase", issueNumber)
		return nil
	}

	// 処理中かチェック
	if a.stateManager.IsProcessing(issueNumber) {
		return fmt.Errorf("issue #%d is already processing", issueNumber)
	}

	// 処理開始
	a.stateManager.SetState(issueNumber, types.IssueStateImplementation, types.IssueStatusProcessing)

	// ラベル遷移（status:ready → status:implementing）
	if err := a.labelManager.TransitionLabel(ctx, int(issueNumber), "status:ready", "status:implementing"); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateImplementation)
		return fmt.Errorf("failed to transition label: %w", err)
	}

	// tmuxウィンドウ作成
	if err := a.tmuxClient.CreateWindowForIssue(a.sessionName, int(issueNumber), "implement"); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateImplementation)
		return fmt.Errorf("failed to create tmux window: %w", err)
	}

	// mainブランチを最新化
	log.Printf("Updating main branch for issue #%d", issueNumber)
	if err := a.worktreeManager.UpdateMainBranch(ctx); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateImplementation)
		return fmt.Errorf("failed to update main branch: %w", err)
	}

	// worktreeを作成（Implementationフェーズ用の独立したworktree）
	log.Printf("Creating worktree for issue #%d", issueNumber)
	if err := a.worktreeManager.CreateWorktree(ctx, int(issueNumber), git.PhaseImplementation); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateImplementation)
		return fmt.Errorf("failed to create worktree: %w", err)
	}
	worktreePath := a.worktreeManager.GetWorktreePath(int(issueNumber), git.PhaseImplementation)
	log.Printf("Worktree created at: %s", worktreePath)

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

	// tmuxウィンドウ内でClaude実行
	windowName := fmt.Sprintf("%d-implement", issueNumber)
	log.Printf("Executing Claude in tmux window for issue #%d", issueNumber)
	if err := a.claudeExecutor.ExecuteInTmux(ctx, phaseConfig, templateVars, a.sessionName, windowName, worktreePath); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateImplementation)
		return fmt.Errorf("failed to execute claude: %w", err)
	}

	// 処理完了
	a.stateManager.MarkAsCompleted(issueNumber, types.IssueStateImplementation)
	log.Printf("Successfully completed implementation action for issue #%d", issueNumber)

	return nil
}

// CanExecute は実装フェーズのアクションが実行可能かを判定する
func (a *ImplementationAction) CanExecute(issue *github.Issue) bool {
	return hasLabel(issue, "status:ready")
}
