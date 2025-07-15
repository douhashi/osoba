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

// ReviewAction はレビューフェーズのアクション実装
type ReviewAction struct {
	types.BaseAction
	sessionName     string
	tmuxClient      TmuxClient
	stateManager    StateManager
	labelManager    LabelManager
	worktreeManager git.WorktreeManager
	claudeExecutor  claude.ClaudeExecutor
	claudeConfig    *claude.ClaudeConfig
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

// Execute はレビューフェーズのアクションを実行する
func (a *ReviewAction) Execute(ctx context.Context, issue *github.Issue) error {
	if issue == nil || issue.Number == nil {
		return fmt.Errorf("invalid issue")
	}

	issueNumber := int64(*issue.Number)
	log.Printf("Executing review action for issue #%d", issueNumber)

	// 既に処理済みかチェック
	if a.stateManager.HasBeenProcessed(issueNumber, types.IssueStateReview) {
		log.Printf("Issue #%d has already been processed for review phase", issueNumber)
		return nil
	}

	// 処理中かチェック
	if a.stateManager.IsProcessing(issueNumber) {
		return fmt.Errorf("issue #%d is already processing", issueNumber)
	}

	// 処理開始
	a.stateManager.SetState(issueNumber, types.IssueStateReview, types.IssueStatusProcessing)

	// ラベル遷移（status:review-requested → status:reviewing）
	if err := a.labelManager.TransitionLabel(ctx, int(issueNumber), "status:review-requested", "status:reviewing"); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateReview)
		return fmt.Errorf("failed to transition label: %w", err)
	}

	// tmuxウィンドウへの切り替え（既存のウィンドウを使用）
	if err := a.tmuxClient.SwitchToIssueWindow(a.sessionName, int(issueNumber)); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateReview)
		return fmt.Errorf("failed to switch tmux window: %w", err)
	}

	// mainブランチを最新化
	log.Printf("Updating main branch for issue #%d", issueNumber)
	if err := a.worktreeManager.UpdateMainBranch(ctx); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateReview)
		return fmt.Errorf("failed to update main branch: %w", err)
	}

	// 既存のworktreeを確認して使用
	var worktreePath string
	phases := []git.Phase{git.PhaseImplementation, git.PhasePlan}
	for _, phase := range phases {
		exists, err := a.worktreeManager.WorktreeExists(ctx, int(issueNumber), phase)
		if err != nil {
			a.stateManager.MarkAsFailed(issueNumber, types.IssueStateReview)
			return fmt.Errorf("failed to check worktree existence: %w", err)
		}
		if exists {
			worktreePath = a.worktreeManager.GetWorktreePath(int(issueNumber), phase)
			log.Printf("Using existing worktree at: %s", worktreePath)
			break
		}
	}

	if worktreePath == "" {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateReview)
		return fmt.Errorf("no worktree found for issue #%d", issueNumber)
	}

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
	windowName := fmt.Sprintf("issue-%d", issueNumber)
	log.Printf("Executing Claude in tmux window for issue #%d", issueNumber)
	if err := a.claudeExecutor.ExecuteInTmux(ctx, phaseConfig, templateVars, a.sessionName, windowName, worktreePath); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, types.IssueStateReview)
		return fmt.Errorf("failed to execute claude: %w", err)
	}

	// レビュー完了後、status:completedラベルを追加
	if err := a.labelManager.AddLabel(ctx, int(issueNumber), "status:completed"); err != nil {
		log.Printf("Warning: failed to add completed label: %v", err)
		// 完了ラベルの追加に失敗してもエラーとしない
	}

	// 処理完了
	a.stateManager.MarkAsCompleted(issueNumber, types.IssueStateReview)
	log.Printf("Successfully completed review action for issue #%d", issueNumber)

	return nil
}

// CanExecute はレビューフェーズのアクションが実行可能かを判定する
func (a *ReviewAction) CanExecute(issue *github.Issue) bool {
	return hasLabel(issue, "status:review-requested")
}
