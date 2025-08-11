package watcher

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// LabelTransitionMetrics はラベル遷移処理のメトリクスを管理する構造体
type LabelTransitionMetrics struct {
	mu                    sync.RWMutex
	TotalTransitions      int64            // 総遷移試行回数
	SuccessfulTransitions int64            // 成功した遷移数
	FailedTransitions     int64            // 失敗した遷移数
	FailureReasons        map[string]int64 // 失敗理由別の回数
	TransitionTypes       map[string]int64 // 遷移パターン別の回数
	StartTime             time.Time        // 開始時刻
	LastTransitionTime    time.Time        // 最後の遷移試行時刻
}

// TransitionType は遷移パターンとその発生回数を表す構造体
type TransitionType struct {
	Type  string // 遷移パターン (例: "status:needs-plan->status:planning")
	Count int64  // 発生回数
}

// NewLabelTransitionMetrics は新しいLabelTransitionMetricsを作成する
func NewLabelTransitionMetrics() *LabelTransitionMetrics {
	return &LabelTransitionMetrics{
		TotalTransitions:      0,
		SuccessfulTransitions: 0,
		FailedTransitions:     0,
		FailureReasons:        make(map[string]int64),
		TransitionTypes:       make(map[string]int64),
		StartTime:             time.Now(),
		LastTransitionTime:    time.Time{},
	}
}

// RecordSuccess は成功したラベル遷移を記録する
func (m *LabelTransitionMetrics) RecordSuccess(issueNumber int, transitionType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalTransitions++
	m.SuccessfulTransitions++
	m.TransitionTypes[transitionType]++
	m.LastTransitionTime = time.Now()
}

// RecordFailure は失敗したラベル遷移を記録する
func (m *LabelTransitionMetrics) RecordFailure(issueNumber int, transitionType string, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalTransitions++
	m.FailedTransitions++
	m.FailureReasons[reason]++
	m.TransitionTypes[transitionType]++
	m.LastTransitionTime = time.Now()
}

// GetSuccessRate は成功率を百分率で返す
func (m *LabelTransitionMetrics) GetSuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.TotalTransitions == 0 {
		return 0.0
	}

	return float64(m.SuccessfulTransitions) / float64(m.TotalTransitions) * 100.0
}

// GetSuccessRateFormatted はフォーマットされた成功率文字列を返す
func (m *LabelTransitionMetrics) GetSuccessRateFormatted() string {
	return fmt.Sprintf("%.2f%%", m.GetSuccessRate())
}

// GetTopFailureReasons は上位N個の失敗理由を取得する
func (m *LabelTransitionMetrics) GetTopFailureReasons(limit int) []FailureReason {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.FailureReasons) == 0 {
		return []FailureReason{}
	}

	// 失敗理由をスライスに変換
	reasons := make([]FailureReason, 0, len(m.FailureReasons))
	for reason, count := range m.FailureReasons {
		reasons = append(reasons, FailureReason{
			Reason: reason,
			Count:  count,
		})
	}

	// 発生回数でソート（降順）
	sort.Slice(reasons, func(i, j int) bool {
		return reasons[i].Count > reasons[j].Count
	})

	// 指定された上限まで返す
	if limit < len(reasons) {
		return reasons[:limit]
	}

	return reasons
}

// GetMostFrequentTransitions は上位N個の頻出遷移パターンを取得する
func (m *LabelTransitionMetrics) GetMostFrequentTransitions(limit int) []TransitionType {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.TransitionTypes) == 0 {
		return []TransitionType{}
	}

	// 遷移パターンをスライスに変換
	transitions := make([]TransitionType, 0, len(m.TransitionTypes))
	for transType, count := range m.TransitionTypes {
		transitions = append(transitions, TransitionType{
			Type:  transType,
			Count: count,
		})
	}

	// 発生回数でソート（降順）
	sort.Slice(transitions, func(i, j int) bool {
		return transitions[i].Count > transitions[j].Count
	})

	// 指定された上限まで返す
	if limit < len(transitions) {
		return transitions[:limit]
	}

	return transitions
}

// Reset はメトリクスをリセットする
func (m *LabelTransitionMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalTransitions = 0
	m.SuccessfulTransitions = 0
	m.FailedTransitions = 0
	m.FailureReasons = make(map[string]int64)
	m.TransitionTypes = make(map[string]int64)
	m.StartTime = time.Now()
	m.LastTransitionTime = time.Time{}
}

// GetUptimeDuration は稼働時間を返す
func (m *LabelTransitionMetrics) GetUptimeDuration() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return time.Since(m.StartTime)
}

// GetSnapshot はメトリクスのスナップショットを返す（読み取り専用）
func (m *LabelTransitionMetrics) GetSnapshot() LabelTransitionMetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// FailureReasonsのコピーを作成
	failureReasons := make(map[string]int64)
	for k, v := range m.FailureReasons {
		failureReasons[k] = v
	}

	// TransitionTypesのコピーを作成
	transitionTypes := make(map[string]int64)
	for k, v := range m.TransitionTypes {
		transitionTypes[k] = v
	}

	return LabelTransitionMetricsSnapshot{
		TotalTransitions:      m.TotalTransitions,
		SuccessfulTransitions: m.SuccessfulTransitions,
		FailedTransitions:     m.FailedTransitions,
		FailureReasons:        failureReasons,
		TransitionTypes:       transitionTypes,
		StartTime:             m.StartTime,
		LastTransitionTime:    m.LastTransitionTime,
		SuccessRate:           m.getSuccessRateUnsafe(),
		UptimeDuration:        time.Since(m.StartTime),
	}
}

// getSuccessRateUnsafe は内部用の成功率計算（ロックなし）
func (m *LabelTransitionMetrics) getSuccessRateUnsafe() float64 {
	if m.TotalTransitions == 0 {
		return 0.0
	}
	return float64(m.SuccessfulTransitions) / float64(m.TotalTransitions) * 100.0
}

// LabelTransitionMetricsSnapshot はメトリクスの読み取り専用スナップショット
type LabelTransitionMetricsSnapshot struct {
	TotalTransitions      int64
	SuccessfulTransitions int64
	FailedTransitions     int64
	FailureReasons        map[string]int64
	TransitionTypes       map[string]int64
	StartTime             time.Time
	LastTransitionTime    time.Time
	SuccessRate           float64
	UptimeDuration        time.Duration
}

// GetSuccessRateFormatted はフォーマットされた成功率文字列を返す
func (s LabelTransitionMetricsSnapshot) GetSuccessRateFormatted() string {
	return fmt.Sprintf("%.2f%%", s.SuccessRate)
}

// GetTopFailureReasons は上位N個の失敗理由を取得する
func (s LabelTransitionMetricsSnapshot) GetTopFailureReasons(limit int) []FailureReason {
	if len(s.FailureReasons) == 0 {
		return []FailureReason{}
	}

	// 失敗理由をスライスに変換
	reasons := make([]FailureReason, 0, len(s.FailureReasons))
	for reason, count := range s.FailureReasons {
		reasons = append(reasons, FailureReason{
			Reason: reason,
			Count:  count,
		})
	}

	// 発生回数でソート（降順）
	sort.Slice(reasons, func(i, j int) bool {
		return reasons[i].Count > reasons[j].Count
	})

	// 指定された上限まで返す
	if limit < len(reasons) {
		return reasons[:limit]
	}

	return reasons
}

// GetMostFrequentTransitions は上位N個の頻出遷移パターンを取得する
func (s LabelTransitionMetricsSnapshot) GetMostFrequentTransitions(limit int) []TransitionType {
	if len(s.TransitionTypes) == 0 {
		return []TransitionType{}
	}

	// 遷移パターンをスライスに変換
	transitions := make([]TransitionType, 0, len(s.TransitionTypes))
	for transType, count := range s.TransitionTypes {
		transitions = append(transitions, TransitionType{
			Type:  transType,
			Count: count,
		})
	}

	// 発生回数でソート（降順）
	sort.Slice(transitions, func(i, j int) bool {
		return transitions[i].Count > transitions[j].Count
	})

	// 指定された上限まで返す
	if limit < len(transitions) {
		return transitions[:limit]
	}

	return transitions
}
