package actions

import (
	"context"
	"fmt"
	"log"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/douhashi/osoba/internal/types"
)

// ImplementationAction は実装フェーズのアクション実装
type ImplementationAction struct {
	types.BaseAction
	sessionName       string
	tmuxClient        TmuxClient
	stateManager      StateManager
	labelManager      LabelManager
	phaseTransitioner PhaseTransitioner
	worktreeManager   git.WorktreeManager
	claudeExecutor    claude.ClaudeExecutor
	claudeConfig      *claude.ClaudeConfig
	logger            logger.Logger
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

// NewImplementationActionWithPhaseTransitioner は新しいImplementationActionをPhaseTransitionerと共に作成する
func NewImplementationActionWithPhaseTransitioner(
	sessionName string,
	tmuxClient TmuxClient,
	stateManager StateManager,
	labelManager LabelManager,
	phaseTransitioner PhaseTransitioner,
	worktreeManager git.WorktreeManager,
	claudeExecutor claude.ClaudeExecutor,
	claudeConfig *claude.ClaudeConfig,
) *ImplementationAction {
	return &ImplementationAction{
		BaseAction:        types.BaseAction{Type: types.ActionTypeImplementation},
		sessionName:       sessionName,
		tmuxClient:        tmuxClient,
		stateManager:      stateManager,
		labelManager:      labelManager,
		phaseTransitioner: phaseTransitioner,
		worktreeManager:   worktreeManager,
		claudeExecutor:    claudeExecutor,
		claudeConfig:      claudeConfig,
	}
}

// NewImplementationActionWithLogger はloggerを注入したImplementationActionを作成する
func NewImplementationActionWithLogger(
	sessionName string,
	tmuxClient TmuxClient,
	stateManager StateManager,
	labelManager LabelManager,
	worktreeManager git.WorktreeManager,
	claudeExecutor claude.ClaudeExecutor,
	claudeConfig *claude.ClaudeConfig,
	logger logger.Logger,
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
		logger:          logger,
	}
}

// logInfo はloggerが設定されている場合は構造化ログを、設定されていない場合は標準ログを出力する
func (a *ImplementationAction) logInfo(msg string, keysAndValues ...interface{}) {
	if a.logger != nil {
		a.logger.Info(msg, keysAndValues...)
	} else {
		// 後方互換性のため、標準ログ出力を維持
		if len(keysAndValues) >= 2 {
			// 特別なケースの処理
			var issueNumber interface{}
			var path interface{}
			for i := 0; i < len(keysAndValues); i += 2 {
				if keysAndValues[i] == "issue_number" {
					issueNumber = keysAndValues[i+1]
				} else if keysAndValues[i] == "path" {
					path = keysAndValues[i+1]
				}
			}

			// pathとissue_numberがある場合
			if path != nil && msg == "Worktree created" {
				log.Printf("%s at: %v", msg, path)
				return
			}

			// issue_numberがある場合は既存のフォーマットを使用
			if issueNumber != nil {
				log.Printf("%s for issue #%v", msg, issueNumber)
				return
			}
		}
		log.Print(msg)
	}
}

// Execute は実装フェーズのアクションを実行する
func (a *ImplementationAction) Execute(ctx context.Context, issue *github.Issue) error {
	if issue == nil || issue.Number == nil {
		return fmt.Errorf("invalid issue")
	}

	issueNumber := int64(*issue.Number)
	a.logInfo("Executing implementation action", "issue_number", issueNumber)

	// 既に処理済みかチェック
	if a.stateManager.HasBeenProcessed(issueNumber, types.IssueStateImplementation) {
		a.logInfo("Issue has already been processed for implementation phase", "issue_number", issueNumber)
		return nil
	}

	// 処理中かチェック
	if a.stateManager.IsProcessing(issueNumber) {
		return fmt.Errorf("issue #%d is already processing", issueNumber)
	}

	// 処理開始
	a.stateManager.SetState(issueNumber, types.IssueStateImplementation, types.IssueStatusProcessing)

	// tmuxウィンドウ作成
	if err := a.tmuxClient.CreateWindowForIssue(a.sessionName, int(issueNumber), "implement"); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateImplementation)
		return fmt.Errorf("failed to create tmux window: %w", err)
	}

	// mainブランチを最新化
	a.logInfo("Updating main branch", "issue_number", issueNumber)
	if err := a.worktreeManager.UpdateMainBranch(ctx); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateImplementation)
		return fmt.Errorf("failed to update main branch: %w", err)
	}

	// worktreeを作成（Implementationフェーズ用の独立したworktree）
	a.logInfo("Creating worktree", "issue_number", issueNumber, "phase", "implementation")
	if err := a.worktreeManager.CreateWorktree(ctx, int(issueNumber), git.PhaseImplementation); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateImplementation)
		return fmt.Errorf("failed to create worktree: %w", err)
	}
	worktreePath := a.worktreeManager.GetWorktreePath(int(issueNumber), git.PhaseImplementation)
	a.logInfo("Worktree created", "issue_number", issueNumber, "path", worktreePath, "phase", "implementation")

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
	a.logInfo("Executing Claude in tmux window", "issue_number", issueNumber, "window_name", windowName, "phase", "implementation")
	if err := a.claudeExecutor.ExecuteInTmux(ctx, phaseConfig, templateVars, a.sessionName, windowName, worktreePath); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateImplementation)
		return fmt.Errorf("failed to execute claude: %w", err)
	}

	// 処理完了
	a.stateManager.MarkAsCompleted(issueNumber, types.IssueStateImplementation)
	a.logInfo("Successfully completed implementation action", "issue_number", issueNumber)

	return nil
}

// CanExecute は実装フェーズのアクションが実行可能かを判定する
func (a *ImplementationAction) CanExecute(issue *github.Issue) bool {
	return hasLabel(issue, "status:ready")
}
