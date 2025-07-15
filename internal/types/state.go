package types

import "time"

// IssuePhase はIssueの処理フェーズを表す型
type IssuePhase string

const (
	IssueStatePlan           IssuePhase = "plan"
	IssueStateImplementation IssuePhase = "implementation"
	IssueStateReview         IssuePhase = "review"
)

// IssueStatus はIssueの処理状態を表す型
type IssueStatus string

const (
	IssueStatusPending    IssueStatus = "pending"
	IssueStatusProcessing IssueStatus = "processing"
	IssueStatusCompleted  IssueStatus = "completed"
	IssueStatusFailed     IssueStatus = "failed"
)

// IssueState はIssueの処理状態を表す構造体
type IssueState struct {
	IssueNumber int64
	Phase       IssuePhase
	LastAction  time.Time
	Status      IssueStatus
}
