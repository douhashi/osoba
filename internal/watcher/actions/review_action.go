package actions

import (
	"context"
	"fmt"
	"log"

	"github.com/douhashi/osoba/internal/watcher"
	"github.com/google/go-github/v67/github"
)

// ReviewAction はレビューフェーズのアクション実装
type ReviewAction struct {
	watcher.BaseAction
	sessionName   string
	tmuxClient    TmuxClient
	stateManager  StateManager
	labelManager  LabelManager
	gitManager    GitManager
	claudeManager ClaudeManager
}

// NewReviewAction は新しいReviewActionを作成する
func NewReviewAction(
	sessionName string,
	tmuxClient TmuxClient,
	stateManager StateManager,
	labelManager LabelManager,
	gitManager GitManager,
	claudeManager ClaudeManager,
) *ReviewAction {
	return &ReviewAction{
		BaseAction:    watcher.BaseAction{Type: watcher.ActionTypeReview},
		sessionName:   sessionName,
		tmuxClient:    tmuxClient,
		stateManager:  stateManager,
		labelManager:  labelManager,
		gitManager:    gitManager,
		claudeManager: claudeManager,
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
	if a.stateManager.HasBeenProcessed(issueNumber, watcher.IssueStateReview) {
		log.Printf("Issue #%d has already been processed for review phase", issueNumber)
		return nil
	}

	// 処理中かチェック
	if a.stateManager.IsProcessing(issueNumber) {
		return fmt.Errorf("issue #%d is already processing", issueNumber)
	}

	// 処理開始
	a.stateManager.SetState(issueNumber, watcher.IssueStateReview, watcher.IssueStatusProcessing)

	// ラベル遷移（status:review-requested → status:reviewing）
	if err := a.labelManager.TransitionLabel(ctx, int(issueNumber), "status:review-requested", "status:reviewing"); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, watcher.IssueStateReview)
		return fmt.Errorf("failed to transition label: %w", err)
	}

	// tmuxウィンドウへの切り替え（既存のウィンドウを使用）
	if err := a.tmuxClient.SwitchToIssueWindow(a.sessionName, int(issueNumber)); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, watcher.IssueStateReview)
		return fmt.Errorf("failed to switch tmux window: %w", err)
	}

	// 既存のworktreeパスを使用（計画フェーズで作成済み）
	workdir := fmt.Sprintf("/tmp/osoba/worktree/%d", issueNumber)

	// claudeプロンプト実行
	if err := a.claudeManager.ExecuteReviewPrompt(ctx, int(issueNumber), workdir); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, watcher.IssueStateReview)
		return fmt.Errorf("failed to execute claude prompt: %w", err)
	}

	// レビュー完了後、status:completedラベルを追加
	if err := a.labelManager.AddLabel(ctx, int(issueNumber), "status:completed"); err != nil {
		log.Printf("Warning: failed to add completed label: %v", err)
		// 完了ラベルの追加に失敗してもエラーとしない
	}

	// 処理完了
	a.stateManager.MarkAsCompleted(issueNumber, watcher.IssueStateReview)
	log.Printf("Successfully completed review action for issue #%d", issueNumber)

	return nil
}

// CanExecute はレビューフェーズのアクションが実行可能かを判定する
func (a *ReviewAction) CanExecute(issue *github.Issue) bool {
	return hasLabel(issue, "status:review-requested")
}
