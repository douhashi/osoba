package watcher

import (
	"github.com/douhashi/osoba/internal/types"
	"github.com/douhashi/osoba/internal/watcher/actions"
)

// StateManagerAdapter はIssueStateManagerをactions.StateManagerインターフェースに適応させるアダプター
type StateManagerAdapter struct {
	manager *IssueStateManager
}

// NewStateManagerAdapter は新しいStateManagerAdapterを作成する
func NewStateManagerAdapter(manager *IssueStateManager) actions.StateManager {
	return &StateManagerAdapter{
		manager: manager,
	}
}

// GetState は指定されたIssueの状態を取得する
func (a *StateManagerAdapter) GetState(issueNumber int64) (*types.IssueState, bool) {
	return a.manager.GetIssueState(issueNumber)
}

// SetState は指定されたIssueの状態を設定する
func (a *StateManagerAdapter) SetState(issueNumber int64, phase types.IssuePhase, status types.IssueStatus) {
	a.manager.SetState(issueNumber, phase, status)
}

// IsProcessing は指定されたIssueが処理中かを確認する
func (a *StateManagerAdapter) IsProcessing(issueNumber int64) bool {
	return a.manager.IsProcessing(issueNumber)
}

// HasBeenProcessed は指定されたIssueの特定フェーズが処理済みかを確認する
func (a *StateManagerAdapter) HasBeenProcessed(issueNumber int64, phase types.IssuePhase) bool {
	return a.manager.HasBeenProcessed(issueNumber, phase)
}

// MarkAsCompleted は指定されたIssueのフェーズを完了状態にする
func (a *StateManagerAdapter) MarkAsCompleted(issueNumber int64, phase types.IssuePhase) {
	a.manager.MarkAsCompleted(issueNumber, phase)
}

// MarkAsFailed は指定されたIssueのフェーズを失敗状態にする
func (a *StateManagerAdapter) MarkAsFailed(issueNumber int64, phase types.IssuePhase) {
	a.manager.MarkAsFailed(issueNumber, phase)
}
