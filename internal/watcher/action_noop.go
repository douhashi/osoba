package watcher

import (
	"context"
	"fmt"

	gh "github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
)

// NoOpAction は何もしないアクション（status:requires-changes用）
type NoOpAction struct {
	logger logger.Logger
}

// NewNoOpAction は新しいNoOpActionを作成する
func NewNoOpAction(logger logger.Logger) *NoOpAction {
	return &NoOpAction{
		logger: logger,
	}
}

// Execute はアクションを実行する（何もしない）
func (a *NoOpAction) Execute(ctx context.Context, issue *gh.Issue) error {
	if issue != nil && issue.Number != nil {
		a.logger.Info("NoOpAction: Skipping action for issue",
			"issueNumber", *issue.Number,
			"reason", "No action required for status:requires-changes label")
	}
	// 常に成功を返す
	return nil
}

// CanExecute はアクションが実行可能かどうかを判定する
func (a *NoOpAction) CanExecute(issue *gh.Issue) bool {
	if issue == nil || issue.Number == nil {
		return false
	}
	return true
}

// GetName はアクション名を返す
func (a *NoOpAction) GetName() string {
	return "NoOpAction"
}

// String はアクションの文字列表現を返す
func (a *NoOpAction) String() string {
	return fmt.Sprintf("NoOpAction")
}
