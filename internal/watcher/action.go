package watcher

import (
	"context"
	"fmt"
	"log"

	"github.com/douhashi/osoba/internal/github"
)

// ActionExecutor はIssueに対するアクションを実行するインターフェース
type ActionExecutor interface {
	Execute(ctx context.Context, issue *github.Issue) error
	CanExecute(issue *github.Issue) bool
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

// ActionManager はアクション実行を管理する構造体
type ActionManager struct {
	sessionName   string
	actionFactory ActionFactory
}

// NewActionManager は新しいActionManagerを作成する
func NewActionManager(sessionName string) *ActionManager {
	return &ActionManager{
		sessionName: sessionName,
	}
}

// SetActionFactory はActionFactoryを設定する
func (m *ActionManager) SetActionFactory(factory ActionFactory) {
	m.actionFactory = factory
}

// ExecuteAction はIssueに対して適切なアクションを実行する
func (m *ActionManager) ExecuteAction(ctx context.Context, issue *github.Issue) error {
	if issue == nil || issue.Number == nil {
		return fmt.Errorf("invalid issue")
	}

	log.Printf("[DEBUG] ActionManager.ExecuteAction called for issue #%d", *issue.Number)

	action := m.GetActionForIssue(issue)
	if action == nil {
		log.Printf("[DEBUG] No action found for issue #%d", *issue.Number)
		return fmt.Errorf("no action found for issue #%d", *issue.Number)
	}

	log.Printf("[DEBUG] Action found for issue #%d: %T", *issue.Number, action)

	if !action.CanExecute(issue) {
		log.Printf("[DEBUG] Action cannot be executed for issue #%d", *issue.Number)
		return fmt.Errorf("action cannot be executed for issue #%d", *issue.Number)
	}

	log.Printf("[DEBUG] Executing action for issue #%d", *issue.Number)
	return action.Execute(ctx, issue)
}

// GetActionForIssue はIssueのラベルに基づいて適切なアクションを返す
func (m *ActionManager) GetActionForIssue(issue *github.Issue) ActionExecutor {
	if m.actionFactory == nil {
		log.Printf("[DEBUG] ActionFactory is nil")
		return nil
	}

	log.Printf("[DEBUG] GetActionForIssue called for issue #%d", *issue.Number)

	// ラベルを取得してログ出力
	labels := []string{}
	for _, label := range issue.Labels {
		if label.Name != nil {
			labels = append(labels, *label.Name)
		}
	}
	log.Printf("[DEBUG] Issue #%d has labels: %v", *issue.Number, labels)

	// ラベルを確認して適切なアクションを返す
	if hasLabel(issue, "status:needs-plan") {
		log.Printf("[DEBUG] Issue #%d has status:needs-plan label, creating PlanAction", *issue.Number)
		return m.actionFactory.CreatePlanAction()
	}
	if hasLabel(issue, "status:ready") {
		log.Printf("[DEBUG] Issue #%d has status:ready label, creating ImplementationAction", *issue.Number)
		return m.actionFactory.CreateImplementationAction()
	}
	if hasLabel(issue, "status:review-requested") {
		log.Printf("[DEBUG] Issue #%d has status:review-requested label, creating ReviewAction", *issue.Number)
		return m.actionFactory.CreateReviewAction()
	}
	if hasLabel(issue, "status:requires-changes") {
		log.Printf("[DEBUG] Issue #%d has status:requires-changes label, creating ReviseAction", *issue.Number)
		return m.actionFactory.CreateReviseAction()
	}

	log.Printf("[DEBUG] No matching label found for issue #%d", *issue.Number)
	return nil
}
