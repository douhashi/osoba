package actions

import "github.com/douhashi/osoba/internal/types"

// StateManagerV2 はV2アクション用の状態管理インターフェース
type StateManagerV2 interface {
	SetState(issueNumber int64, phase types.IssuePhase, status types.IssueStatus)
	GetState(issueNumber int64, phase types.IssuePhase) types.IssueStatus
	HasBeenProcessed(issueNumber int64, phase types.IssuePhase) bool
	IsProcessing(issueNumber int64) bool
	MarkAsCompleted(issueNumber int64, phase types.IssuePhase)
	MarkAsFailed(issueNumber int64, phase types.IssuePhase)
	Clear(issueNumber int64)
	GetAllStates() map[int64]map[types.IssuePhase]types.IssueStatus
}
