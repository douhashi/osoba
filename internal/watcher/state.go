package watcher

import (
	"sync"
	"time"

	"github.com/douhashi/osoba/internal/types"
)

// IssueStateManager はIssueの状態を管理する構造体
type IssueStateManager struct {
	mu     sync.RWMutex
	states map[int64]*types.IssueState
}

// NewIssueStateManager は新しいIssueStateManagerを作成する
func NewIssueStateManager() *IssueStateManager {
	return &IssueStateManager{
		states: make(map[int64]*types.IssueState),
	}
}

// GetIssueState は指定されたIssueの状態を取得する（旧インターフェース）
func (m *IssueStateManager) GetIssueState(issueNumber int64) (*types.IssueState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.states[issueNumber]
	if !exists {
		return nil, false
	}

	// 状態のコピーを返す（安全のため）
	stateCopy := &types.IssueState{
		IssueNumber: state.IssueNumber,
		Phase:       state.Phase,
		LastAction:  state.LastAction,
		Status:      state.Status,
	}

	return stateCopy, true
}

// SetState は指定されたIssueの状態を設定する
func (m *IssueStateManager) SetState(issueNumber int64, phase types.IssuePhase, status types.IssueStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.states[issueNumber] = &types.IssueState{
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

	return state.Status == types.IssueStatusProcessing
}

// HasBeenProcessed は指定されたIssueの特定フェーズが処理済みかを確認する
func (m *IssueStateManager) HasBeenProcessed(issueNumber int64, phase types.IssuePhase) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.states[issueNumber]
	if !exists {
		return false
	}

	return state.Phase == phase && state.Status == types.IssueStatusCompleted
}

// MarkAsCompleted は指定されたIssueのフェーズを完了状態にする
func (m *IssueStateManager) MarkAsCompleted(issueNumber int64, phase types.IssuePhase) {
	m.SetState(issueNumber, phase, types.IssueStatusCompleted)
}

// MarkAsFailed は指定されたIssueのフェーズを失敗状態にする
func (m *IssueStateManager) MarkAsFailed(issueNumber int64, phase types.IssuePhase) {
	m.SetState(issueNumber, phase, types.IssueStatusFailed)
}

// GetAllIssueStates はすべてのIssueの状態を取得する（旧インターフェース）
func (m *IssueStateManager) GetAllIssueStates() map[int64]*types.IssueState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 状態のコピーを作成
	statesCopy := make(map[int64]*types.IssueState)
	for k, v := range m.states {
		statesCopy[k] = &types.IssueState{
			IssueNumber: v.IssueNumber,
			Phase:       v.Phase,
			LastAction:  v.LastAction,
			Status:      v.Status,
		}
	}

	return statesCopy
}

// Clear は指定されたIssueのすべての状態をクリアする
func (m *IssueStateManager) Clear(issueNumber int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.states, issueNumber)
}

// GetState はStateManagerV2インターフェース互換のメソッド
func (m *IssueStateManager) GetState(issueNumber int64, phase types.IssuePhase) types.IssueStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.states[issueNumber]
	if !exists {
		return types.IssueStatusPending
	}

	// 指定されたフェーズの状態を返す
	if state.Phase == phase {
		return state.Status
	}

	return types.IssueStatusPending
}

// GetAllStates はStateManagerV2インターフェース互換のメソッド
func (m *IssueStateManager) GetAllStates() map[int64]map[types.IssuePhase]types.IssueStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// フェーズごとの状態を返す
	statesCopy := make(map[int64]map[types.IssuePhase]types.IssueStatus)
	for issueNumber, state := range m.states {
		if _, exists := statesCopy[issueNumber]; !exists {
			statesCopy[issueNumber] = make(map[types.IssuePhase]types.IssueStatus)
		}
		statesCopy[issueNumber][state.Phase] = state.Status
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
			(state.Status == types.IssueStatusCompleted || state.Status == types.IssueStatusFailed) {
			delete(m.states, issueNumber)
		}
	}
}
