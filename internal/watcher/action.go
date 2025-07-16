package watcher

import (
	"context"
	"fmt"

	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/types"
)

// ActionExecutor はIssueに対するアクションを実行するインターフェース
type ActionExecutor interface {
	Execute(ctx context.Context, issue *github.Issue) error
	CanExecute(issue *github.Issue) bool
}

// PlanAction は計画フェーズのアクション実装
type PlanAction struct {
	types.BaseAction
}

// NewPlanAction は新しいPlanActionを作成する
func NewPlanAction() *PlanAction {
	return &PlanAction{
		BaseAction: types.BaseAction{Type: types.ActionTypePlan},
	}
}

// Execute は計画フェーズのアクションを実行する
func (a *PlanAction) Execute(ctx context.Context, issue *github.Issue) error {
	// TODO: 実際の実装は後で行う
	return nil
}

// CanExecute は計画フェーズのアクションが実行可能かを判定する
func (a *PlanAction) CanExecute(issue *github.Issue) bool {
	return hasLabel(issue, "status:needs-plan")
}

// ImplementationAction は実装フェーズのアクション実装
type ImplementationAction struct {
	types.BaseAction
}

// NewImplementationAction は新しいImplementationActionを作成する
func NewImplementationAction() *ImplementationAction {
	return &ImplementationAction{
		BaseAction: types.BaseAction{Type: types.ActionTypeImplementation},
	}
}

// Execute は実装フェーズのアクションを実行する
func (a *ImplementationAction) Execute(ctx context.Context, issue *github.Issue) error {
	// TODO: 実際の実装は後で行う
	return nil
}

// CanExecute は実装フェーズのアクションが実行可能かを判定する
func (a *ImplementationAction) CanExecute(issue *github.Issue) bool {
	return hasLabel(issue, "status:ready")
}

// ReviewAction はレビューフェーズのアクション実装
type ReviewAction struct {
	types.BaseAction
}

// NewReviewAction は新しいReviewActionを作成する
func NewReviewAction() *ReviewAction {
	return &ReviewAction{
		BaseAction: types.BaseAction{Type: types.ActionTypeReview},
	}
}

// Execute はレビューフェーズのアクションを実行する
func (a *ReviewAction) Execute(ctx context.Context, issue *github.Issue) error {
	// TODO: 実際の実装は後で行う
	return nil
}

// CanExecute はレビューフェーズのアクションが実行可能かを判定する
func (a *ReviewAction) CanExecute(issue *github.Issue) bool {
	return hasLabel(issue, "status:review-requested")
}

// GetActionForIssue はIssueのラベルに基づいて適切なアクションを返す
func GetActionForIssue(issue *github.Issue) ActionExecutor {
	if issue == nil {
		return nil
	}

	// ラベルを確認して適切なアクションを返す
	if hasLabel(issue, "status:needs-plan") {
		return NewPlanAction()
	}
	if hasLabel(issue, "status:ready") {
		return NewImplementationAction()
	}
	if hasLabel(issue, "status:review-requested") {
		return NewReviewAction()
	}

	return nil
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
	stateManager  *IssueStateManager
	actionFactory ActionFactory
}

// NewActionManager は新しいActionManagerを作成する
func NewActionManager(sessionName string) *ActionManager {
	return &ActionManager{
		sessionName:  sessionName,
		stateManager: NewIssueStateManager(),
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
		// 後方互換性のため、factoryがない場合は簡易実装を使用
		return GetActionForIssue(issue)
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

// GetStateManager は状態管理オブジェクトを返す
func (m *ActionManager) GetStateManager() *IssueStateManager {
	return m.stateManager
}
