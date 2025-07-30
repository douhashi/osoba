package watcher

import (
	"context"
	"fmt"

	"github.com/douhashi/osoba/internal/github"
)

// ActionFactory インターフェースは action_factory.go で定義済み

// ActionManagerExtended は拡張されたアクション実行管理構造体
type ActionManagerExtended struct {
	sessionName string
	factory     ActionFactory
}

// NewActionManagerExtended は新しいActionManagerExtendedを作成する
func NewActionManagerExtended(sessionName string, factory ActionFactory) *ActionManagerExtended {
	return &ActionManagerExtended{
		sessionName: sessionName,
		factory:     factory,
	}
}

// ExecuteAction はIssueに対して適切なアクションを実行する
func (m *ActionManagerExtended) ExecuteAction(ctx context.Context, issue *github.Issue) error {
	if issue == nil || issue.Number == nil {
		return fmt.Errorf("invalid issue")
	}

	action := m.GetActionForIssue(issue)
	if action == nil {
		return fmt.Errorf("no action found for issue #%d", *issue.Number)
	}

	if !action.CanExecute(issue) {
		return fmt.Errorf("action cannot be executed for issue #%d", *issue.Number)
	}

	return action.Execute(ctx, issue)
}

// GetActionForIssue はIssueのラベルに基づいて適切なアクションを返す
func (m *ActionManagerExtended) GetActionForIssue(issue *github.Issue) ActionExecutor {
	if issue == nil {
		return nil
	}

	// ラベルを確認して適切なアクションを返す
	if hasLabel(issue, "status:needs-plan") {
		return m.factory.CreatePlanAction()
	}
	if hasLabel(issue, "status:ready") {
		return m.factory.CreateImplementationAction()
	}
	if hasLabel(issue, "status:review-requested") {
		return m.factory.CreateReviewAction()
	}

	return nil
}
