package actions

import (
	"context"
	"fmt"
	"log"

	"github.com/douhashi/osoba/internal/tmux"
	"github.com/douhashi/osoba/internal/watcher"
	"github.com/google/go-github/v67/github"
)

// TmuxClient はtmux操作のインターフェース
type TmuxClient interface {
	CreateWindowForIssue(sessionName string, issueNumber int) error
	SwitchToIssueWindow(sessionName string, issueNumber int) error
	WindowExists(sessionName, windowName string) (bool, error)
}

// StateManager は状態管理のインターフェース
type StateManager interface {
	GetState(issueNumber int64) (*watcher.IssueState, bool)
	SetState(issueNumber int64, phase watcher.IssuePhase, status watcher.IssueStatus)
	IsProcessing(issueNumber int64) bool
	HasBeenProcessed(issueNumber int64, phase watcher.IssuePhase) bool
	MarkAsCompleted(issueNumber int64, phase watcher.IssuePhase)
	MarkAsFailed(issueNumber int64, phase watcher.IssuePhase)
}

// DefaultTmuxClient はデフォルトのtmuxクライアント実装
type DefaultTmuxClient struct{}

func (c *DefaultTmuxClient) CreateWindowForIssue(sessionName string, issueNumber int) error {
	return tmux.CreateWindowForIssue(sessionName, issueNumber)
}

func (c *DefaultTmuxClient) SwitchToIssueWindow(sessionName string, issueNumber int) error {
	return tmux.SwitchToIssueWindow(sessionName, issueNumber)
}

func (c *DefaultTmuxClient) WindowExists(sessionName, windowName string) (bool, error) {
	return tmux.WindowExists(sessionName, windowName)
}

// PlanAction は計画フェーズのアクション実装
type PlanAction struct {
	watcher.BaseAction
	sessionName  string
	tmuxClient   TmuxClient
	stateManager StateManager
}

// NewPlanAction は新しいPlanActionを作成する
func NewPlanAction(sessionName string, tmuxClient TmuxClient, stateManager StateManager) *PlanAction {
	return &PlanAction{
		BaseAction:   watcher.BaseAction{Type: watcher.ActionTypePlan},
		sessionName:  sessionName,
		tmuxClient:   tmuxClient,
		stateManager: stateManager,
	}
}

// Execute は計画フェーズのアクションを実行する
func (a *PlanAction) Execute(ctx context.Context, issue *github.Issue) error {
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

	// tmuxウィンドウ作成
	if err := a.tmuxClient.CreateWindowForIssue(a.sessionName, int(issueNumber)); err != nil {
		a.stateManager.MarkAsFailed(issueNumber, watcher.IssueStatePlan)
		return fmt.Errorf("failed to create tmux window: %w", err)
	}

	// TODO: git worktree作成
	// TODO: claudeプロンプト実行

	// 処理完了
	a.stateManager.MarkAsCompleted(issueNumber, watcher.IssueStatePlan)
	log.Printf("Successfully completed plan action for issue #%d", issueNumber)

	return nil
}

// CanExecute は計画フェーズのアクションが実行可能かを判定する
func (a *PlanAction) CanExecute(issue *github.Issue) bool {
	return hasLabel(issue, "status:needs-plan")
}

// hasLabel はIssueが指定されたラベルを持っているかを確認する
func hasLabel(issue *github.Issue, labelName string) bool {
	if issue == nil || issue.Labels == nil {
		return false
	}

	for _, label := range issue.Labels {
		if label.Name != nil && *label.Name == labelName {
			return true
		}
	}
	return false
}
