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

// ReviewAction はレビューフェーズのアクション実装
type ReviewAction struct {
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

// NewReviewAction は新しいReviewActionを作成する
func NewReviewAction(
	sessionName string,
	tmuxClient TmuxClient,
	stateManager StateManager,
	labelManager LabelManager,
	worktreeManager git.WorktreeManager,
	claudeExecutor claude.ClaudeExecutor,
	claudeConfig *claude.ClaudeConfig,
) *ReviewAction {
	return &ReviewAction{
		BaseAction:      types.BaseAction{Type: types.ActionTypeReview},
		sessionName:     sessionName,
		tmuxClient:      tmuxClient,
		stateManager:    stateManager,
		labelManager:    labelManager,
		worktreeManager: worktreeManager,
		claudeExecutor:  claudeExecutor,
		claudeConfig:    claudeConfig,
	}
}

// NewReviewActionWithPhaseTransitioner は新しいReviewActionをPhaseTransitionerと共に作成する
func NewReviewActionWithPhaseTransitioner(
	sessionName string,
	tmuxClient TmuxClient,
	stateManager StateManager,
	labelManager LabelManager,
	phaseTransitioner PhaseTransitioner,
	worktreeManager git.WorktreeManager,
	claudeExecutor claude.ClaudeExecutor,
	claudeConfig *claude.ClaudeConfig,
) *ReviewAction {
	return &ReviewAction{
		BaseAction:        types.BaseAction{Type: types.ActionTypeReview},
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

// NewReviewActionWithLogger はloggerを注入したReviewActionを作成する
func NewReviewActionWithLogger(
	sessionName string,
	tmuxClient TmuxClient,
	stateManager StateManager,
	labelManager LabelManager,
	worktreeManager git.WorktreeManager,
	claudeExecutor claude.ClaudeExecutor,
	claudeConfig *claude.ClaudeConfig,
	logger logger.Logger,
) *ReviewAction {
	return &ReviewAction{
		BaseAction:      types.BaseAction{Type: types.ActionTypeReview},
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
func (a *ReviewAction) logInfo(msg string, keysAndValues ...interface{}) {
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

// logWarn はloggerが設定されている場合は構造化ログを、設定されていない場合は標準ログを出力する
func (a *ReviewAction) logWarn(msg string, keysAndValues ...interface{}) {
	if a.logger != nil {
		a.logger.Warn(msg, keysAndValues...)
	} else {
		// 後方互換性のため、標準ログ出力を維持
		if len(keysAndValues) >= 2 {
			// errorフィールドを探す
			var err interface{}
			for i := 0; i < len(keysAndValues); i += 2 {
				if keysAndValues[i] == "error" {
					err = keysAndValues[i+1]
					break
				}
			}
			if err != nil {
				log.Printf("Warning: %s: %v", msg, err)
				return
			}
		}
		log.Printf("Warning: %s", msg)
	}
}

// Execute はレビューフェーズのアクションを実行する
func (a *ReviewAction) Execute(ctx context.Context, issue *github.Issue) error {
	if issue == nil || issue.Number == nil {
		return fmt.Errorf("invalid issue")
	}

	issueNumber := int64(*issue.Number)
	a.logInfo("Executing review action", "issue_number", issueNumber)

	// 既に処理済みかチェック
	if a.stateManager.HasBeenProcessed(issueNumber, types.IssueStateReview) {
		a.logInfo("Issue has already been processed for review phase", "issue_number", issueNumber)
		return nil
	}

	// 処理中かチェック
	if a.stateManager.IsProcessing(issueNumber) {
		return fmt.Errorf("issue #%d is already processing", issueNumber)
	}

	// 処理開始
	a.stateManager.SetState(issueNumber, types.IssueStateReview, types.IssueStatusProcessing)

	// tmuxウィンドウ作成
	if err := a.tmuxClient.CreateWindowForIssue(a.sessionName, int(issueNumber), "review"); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateReview)
		return fmt.Errorf("failed to create tmux window: %w", err)
	}

	// mainブランチを最新化
	a.logInfo("Updating main branch", "issue_number", issueNumber)
	if err := a.worktreeManager.UpdateMainBranch(ctx); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateReview)
		return fmt.Errorf("failed to update main branch: %w", err)
	}

	// worktreeを作成（Reviewフェーズ用の独立したworktree）
	a.logInfo("Creating worktree", "issue_number", issueNumber, "phase", "review")
	if err := a.worktreeManager.CreateWorktree(ctx, int(issueNumber), git.PhaseReview); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateReview)
		return fmt.Errorf("failed to create worktree: %w", err)
	}
	worktreePath := a.worktreeManager.GetWorktreePath(int(issueNumber), git.PhaseReview)
	a.logInfo("Worktree created", "issue_number", issueNumber, "path", worktreePath, "phase", "review")

	// Claude実行用の変数を準備
	templateVars := &claude.TemplateVariables{
		IssueNumber: int(issueNumber),
		IssueTitle:  getIssueTitle(issue),
		RepoName:    getRepoName(),
	}

	// Claude設定を取得
	phaseConfig, exists := a.claudeConfig.GetPhase("review")
	if !exists {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateReview)
		return fmt.Errorf("review phase config not found")
	}

	// tmuxウィンドウ内でClaude実行
	windowName := fmt.Sprintf("%d-review", issueNumber)
	a.logInfo("Executing Claude in tmux window", "issue_number", issueNumber, "window_name", windowName, "phase", "review")
	if err := a.claudeExecutor.ExecuteInTmux(ctx, phaseConfig, templateVars, a.sessionName, windowName, worktreePath); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateReview)
		return fmt.Errorf("failed to execute claude: %w", err)
	}

	// レビュー完了後、status:completedラベルを追加
	if err := a.labelManager.AddLabel(ctx, int(issueNumber), "status:completed"); err != nil {
		a.logWarn("failed to add completed label", "issue_number", issueNumber, "error", err)
		// 完了ラベルの追加に失敗してもエラーとしない
	}

	// 処理完了
	a.stateManager.MarkAsCompleted(issueNumber, types.IssueStateReview)
	a.logInfo("Successfully completed review action", "issue_number", issueNumber)

	return nil
}

// CanExecute はレビューフェーズのアクションが実行可能かを判定する
func (a *ReviewAction) CanExecute(issue *github.Issue) bool {
	return hasLabel(issue, "status:review-requested")
}
