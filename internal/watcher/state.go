package watcher

import (
	"sync"
	"time"
)

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

// IssueStateManager はIssueの状態を管理する構造体
type IssueStateManager struct {
	mu     sync.RWMutex
	states map[int64]*IssueState
}

// NewIssueStateManager は新しいIssueStateManagerを作成する
func NewIssueStateManager() *IssueStateManager {
	return &IssueStateManager{
		states: make(map[int64]*IssueState),
	}
}

// GetState は指定されたIssueの状態を取得する
func (m *IssueStateManager) GetState(issueNumber int64) (*IssueState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.states[issueNumber]
	if !exists {
		return nil, false
	}

	// 状態のコピーを返す（安全のため）
	stateCopy := &IssueState{
		IssueNumber: state.IssueNumber,
		Phase:       state.Phase,
		LastAction:  state.LastAction,
		Status:      state.Status,
	}

	return stateCopy, true
}

// SetState は指定されたIssueの状態を設定する
func (m *IssueStateManager) SetState(issueNumber int64, phase IssuePhase, status IssueStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.states[issueNumber] = &IssueState{
		IssueNumber: issueNumber,
		Phase:       phase,
		LastAction:  time.Now(),
		Status:      status,
	}
}

// IsProcessing は指定されたIssueが処理中かを確認する
func (m *IssueStateManager) IsProcessing(issueNumber int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.states[issueNumber]
	if !exists {
		return false
	}

	return state.Status == IssueStatusProcessing
}

// HasBeenProcessed は指定されたIssueの特定フェーズが処理済みかを確認する
func (m *IssueStateManager) HasBeenProcessed(issueNumber int64, phase IssuePhase) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.states[issueNumber]
	if !exists {
		return false
	}

	return state.Phase == phase && state.Status == IssueStatusCompleted
}

// MarkAsCompleted は指定されたIssueのフェーズを完了状態にする
func (m *IssueStateManager) MarkAsCompleted(issueNumber int64, phase IssuePhase) {
	m.SetState(issueNumber, phase, IssueStatusCompleted)
}

// MarkAsFailed は指定されたIssueのフェーズを失敗状態にする
func (m *IssueStateManager) MarkAsFailed(issueNumber int64, phase IssuePhase) {
	m.SetState(issueNumber, phase, IssueStatusFailed)
}

// GetAllStates はすべてのIssueの状態を取得する
func (m *IssueStateManager) GetAllStates() map[int64]*IssueState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 状態のコピーを作成
	statesCopy := make(map[int64]*IssueState)
	for k, v := range m.states {
		statesCopy[k] = &IssueState{
			IssueNumber: v.IssueNumber,
			Phase:       v.Phase,
			LastAction:  v.LastAction,
			Status:      v.Status,
		}
	}

	return statesCopy
}

// CleanupOldStates は古い状態を削除する
func (m *IssueStateManager) CleanupOldStates(olderThan time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	for issueNumber, state := range m.states {
		if state.LastAction.Before(cutoff) &&
			(state.Status == IssueStatusCompleted || state.Status == IssueStatusFailed) {
			delete(m.states, issueNumber)
		}
	}
}
