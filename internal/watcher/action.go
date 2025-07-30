package watcher

import (
	"context"
	"fmt"

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
func (m *ActionManager) GetActionForIssue(issue *github.Issue) ActionExecutor {
	if m.actionFactory == nil {
		return nil
	}

	// ラベルを確認して適切なアクションを返す
	if hasLabel(issue, "status:needs-plan") {
		return m.actionFactory.CreatePlanAction()
	}
	if hasLabel(issue, "status:ready") {
		return m.actionFactory.CreateImplementationAction()
	}
	if hasLabel(issue, "status:review-requested") {
		return m.actionFactory.CreateReviewAction()
	}

	return nil
}
