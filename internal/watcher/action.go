package watcher

import (
	"context"
	"fmt"

	"github.com/google/go-github/v67/github"
)

// ActionExecutor はIssueに対するアクションを実行するインターフェース
type ActionExecutor interface {
	Execute(ctx context.Context, issue *github.Issue) error
	CanExecute(issue *github.Issue) bool
}

// ActionType はアクションの種類を表す型
type ActionType string

const (
	ActionTypePlan           ActionType = "plan"
	ActionTypeImplementation ActionType = "implementation"
	ActionTypeReview         ActionType = "review"
)

// BaseAction はActionExecutorの基本実装
type BaseAction struct {
	Type ActionType
}

// ActionType はアクションの種類を返す
func (a *BaseAction) ActionType() string {
	return string(a.Type)
}

// PlanAction は計画フェーズのアクション実装
type PlanAction struct {
	BaseAction
}

// NewPlanAction は新しいPlanActionを作成する
func NewPlanAction() *PlanAction {
	return &PlanAction{
		BaseAction: BaseAction{Type: ActionTypePlan},
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
	BaseAction
}

// NewImplementationAction は新しいImplementationActionを作成する
func NewImplementationAction() *ImplementationAction {
	return &ImplementationAction{
		BaseAction: BaseAction{Type: ActionTypeImplementation},
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
	BaseAction
}

// NewReviewAction は新しいReviewActionを作成する
func NewReviewAction() *ReviewAction {
	return &ReviewAction{
		BaseAction: BaseAction{Type: ActionTypeReview},
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
	sessionName  string
	stateManager *IssueStateManager
}

// NewActionManager は新しいActionManagerを作成する
func NewActionManager(sessionName string) *ActionManager {
	return &ActionManager{
		sessionName:  sessionName,
		stateManager: NewIssueStateManager(),
	}
}

// ExecuteAction はIssueに対して適切なアクションを実行する
func (m *ActionManager) ExecuteAction(ctx context.Context, issue *github.Issue) error {
	if issue == nil || issue.Number == nil {
		return fmt.Errorf("invalid issue")
	}

	action := m.GetActionForIssue(issue)
	if action == nil {
		return fmt.Errorf("no action found for issue #%d", issue.GetNumber())
	}

	if !action.CanExecute(issue) {
		return fmt.Errorf("action cannot be executed for issue #%d", issue.GetNumber())
	}

	return action.Execute(ctx, issue)
}

// GetActionForIssue はIssueのラベルに基づいて適切なアクションを返す
func (m *ActionManager) GetActionForIssue(issue *github.Issue) ActionExecutor {
	// actions パッケージをインポートして使用する必要があるため、
	// 一時的に簡易実装を使用
	return GetActionForIssue(issue)
}

// GetStateManager は状態管理オブジェクトを返す
func (m *ActionManager) GetStateManager() *IssueStateManager {
	return m.stateManager
}
