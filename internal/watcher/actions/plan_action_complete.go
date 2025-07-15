package actions

import (
	"context"
	"fmt"
	"log"

	"github.com/douhashi/osoba/internal/watcher"
	"github.com/google/go-github/v50/github"
)

// LabelManager はラベル管理のインターフェース
type LabelManager interface {
	TransitionLabel(ctx context.Context, issueNumber int, from, to string) error
	AddLabel(ctx context.Context, issueNumber int, label string) error
	RemoveLabel(ctx context.Context, issueNumber int, label string) error
}

// GitManager はGit操作のインターフェース
type GitManager interface {
	CreateWorktreeForIssue(issueNumber int, branchName string) (string, error)
	DeleteWorktree(path string) error
}

// ClaudeManager はClaude実行のインターフェース
type ClaudeManager interface {
	ExecutePlanPrompt(ctx context.Context, issueNumber int, workdir string) error
	ExecuteImplementationPrompt(ctx context.Context, issueNumber int, workdir string) error
	ExecuteReviewPrompt(ctx context.Context, issueNumber int, workdir string) error
}

// PlanActionComplete は完全な計画フェーズのアクション実装
type PlanActionComplete struct {
	watcher.BaseAction
	sessionName   string
	tmuxClient    TmuxClient
	stateManager  StateManager
	labelManager  LabelManager
	gitManager    GitManager
	claudeManager ClaudeManager
}

// NewPlanActionComplete は新しいPlanActionCompleteを作成する
func NewPlanActionComplete(
	sessionName string,
	tmuxClient TmuxClient,
	stateManager StateManager,
	labelManager LabelManager,
	gitManager GitManager,
	claudeManager ClaudeManager,
) *PlanActionComplete {
	return &PlanActionComplete{
		BaseAction:    watcher.BaseAction{Type: watcher.ActionTypePlan},
		sessionName:   sessionName,
		tmuxClient:    tmuxClient,
		stateManager:  stateManager,
		labelManager:  labelManager,
		gitManager:    gitManager,
		claudeManager: claudeManager,
	}
}

// Execute は計画フェーズのアクションを実行する
func (a *PlanActionComplete) Execute(ctx context.Context, issue *github.Issue) error {
	if issue == nil || issue.Number == nil {
		return fmt.Errorf("invalid issue")
	}

	issueNumber := int64(*issue.Number)
	log.Printf("Executing plan action for issue #%d", issueNumber)

	// 既に処理済みかチェック
	if a.stateManager.HasBeenProcessed(issueNumber, watcher.IssueStatePlan) {
		log.Printf("Issue #%d has already been processed for plan phase", issueNumber)
		return nil
	}

	// 処理中かチェック
	if a.stateManager.IsProcessing(issueNumber) {
		return fmt.Errorf("issue #%d is already processing", issueNumber)
	}

	// 処理開始
	a.stateManager.SetState(issueNumber, watcher.IssueStatePlan, watcher.IssueStatusProcessing)

	// ラベル遷移（status:needs-plan → status:planning）
	if err := a.labelManager.TransitionLabel(ctx, int(issueNumber), "status:needs-plan", "status:planning"); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, watcher.IssueStatePlan)
		return fmt.Errorf("failed to transition label: %w", err)
	}

	// tmuxウィンドウ作成
	if err := a.tmuxClient.CreateWindowForIssue(a.sessionName, int(issueNumber)); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, watcher.IssueStatePlan)
		return fmt.Errorf("failed to create tmux window: %w", err)
	}

	// git worktree作成
	branchName := fmt.Sprintf("feat/#%d-phase-action-execution", issueNumber)
	workdir, err := a.gitManager.CreateWorktreeForIssue(int(issueNumber), branchName)
	if err != nil {
		a.stateManager.MarkAsFailed(issueNumber, watcher.IssueStatePlan)
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// claudeプロンプト実行
	if err := a.claudeManager.ExecutePlanPrompt(ctx, int(issueNumber), workdir); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, watcher.IssueStatePlan)
		return fmt.Errorf("failed to execute claude prompt: %w", err)
	}

	// 処理完了
	a.stateManager.MarkAsCompleted(issueNumber, watcher.IssueStatePlan)
	log.Printf("Successfully completed plan action for issue #%d", issueNumber)

	return nil
}

// CanExecute は計画フェーズのアクションが実行可能かを判定する
func (a *PlanActionComplete) CanExecute(issue *github.Issue) bool {
	return hasLabel(issue, "status:needs-plan")
}
