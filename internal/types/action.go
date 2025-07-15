package types

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
