package actions

import (
	"context"
	"fmt"
	"log"

	"github.com/douhashi/osoba/internal/watcher"
	"github.com/google/go-github/v67/github"
)

// ImplementationAction は実装フェーズのアクション実装
type ImplementationAction struct {
	watcher.BaseAction
	sessionName   string
	tmuxClient    TmuxClient
	stateManager  StateManager
	labelManager  LabelManager
	gitManager    GitManager
	claudeManager ClaudeManager
}

// NewImplementationAction は新しいImplementationActionを作成する
func NewImplementationAction(
	sessionName string,
	tmuxClient TmuxClient,
	stateManager StateManager,
	labelManager LabelManager,
	gitManager GitManager,
	claudeManager ClaudeManager,
) *ImplementationAction {
	return &ImplementationAction{
		BaseAction:    watcher.BaseAction{Type: watcher.ActionTypeImplementation},
		sessionName:   sessionName,
		tmuxClient:    tmuxClient,
		stateManager:  stateManager,
		labelManager:  labelManager,
		gitManager:    gitManager,
		claudeManager: claudeManager,
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
	if a.stateManager.HasBeenProcessed(issueNumber, watcher.IssueStateImplementation) {
		log.Printf("Issue #%d has already been processed for implementation phase", issueNumber)
		return nil
	}

	// 処理中かチェック
	if a.stateManager.IsProcessing(issueNumber) {
		return fmt.Errorf("issue #%d is already processing", issueNumber)
	}

	// 処理開始
	a.stateManager.SetState(issueNumber, watcher.IssueStateImplementation, watcher.IssueStatusProcessing)

	// ラベル遷移（status:ready → status:implementing）
	if err := a.labelManager.TransitionLabel(ctx, int(issueNumber), "status:ready", "status:implementing"); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, watcher.IssueStateImplementation)
		return fmt.Errorf("failed to transition label: %w", err)
	}

	// tmuxウィンドウへの切り替え（既存のウィンドウを使用）
	if err := a.tmuxClient.SwitchToIssueWindow(a.sessionName, int(issueNumber)); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, watcher.IssueStateImplementation)
		return fmt.Errorf("failed to switch tmux window: %w", err)
	}

	// 既存のworktreeパスを使用（計画フェーズで作成済み）
	workdir := fmt.Sprintf("/tmp/osoba/worktree/%d", issueNumber)

	// claudeプロンプト実行
	if err := a.claudeManager.ExecuteImplementationPrompt(ctx, int(issueNumber), workdir); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, watcher.IssueStateImplementation)
		return fmt.Errorf("failed to execute claude prompt: %w", err)
	}

	// 処理完了
	a.stateManager.MarkAsCompleted(issueNumber, watcher.IssueStateImplementation)
	log.Printf("Successfully completed implementation action for issue #%d", issueNumber)

	return nil
}

// CanExecute は実装フェーズのアクションが実行可能かを判定する
func (a *ImplementationAction) CanExecute(issue *github.Issue) bool {
	return hasLabel(issue, "status:ready")
}
