package watcher

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// AutoMergeMetrics は自動マージ処理のメトリクスを管理する構造体
type AutoMergeMetrics struct {
	mu               sync.RWMutex
	TotalAttempts    int64            // 総試行回数
	SuccessfulMerges int64            // 成功したマージ数
	FailedMerges     int64            // 失敗したマージ数
	FailureReasons   map[string]int64 // 失敗理由別の回数
	StartTime        time.Time        // 開始時刻
	LastAttemptTime  time.Time        // 最後の試行時刻
}

// FailureReason は失敗理由とその発生回数を表す構造体
type FailureReason struct {
	Reason string // 失敗理由
	Count  int64  // 発生回数
}

// NewAutoMergeMetrics は新しいAutoMergeMetricsを作成する
func NewAutoMergeMetrics() *AutoMergeMetrics {
	return &AutoMergeMetrics{
		TotalAttempts:    0,
		SuccessfulMerges: 0,
		FailedMerges:     0,
		FailureReasons:   make(map[string]int64),
		StartTime:        time.Now(),
		LastAttemptTime:  time.Time{},
	}
}

// RecordSuccess は成功したマージを記録する
func (m *AutoMergeMetrics) RecordSuccess(issueNumber int, prNumber int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalAttempts++
	m.SuccessfulMerges++
	m.LastAttemptTime = time.Now()
}

// RecordFailure は失敗したマージを記録する
func (m *AutoMergeMetrics) RecordFailure(issueNumber int, prNumber int, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalAttempts++
	m.FailedMerges++
	m.FailureReasons[reason]++
	m.LastAttemptTime = time.Now()
}

// GetSuccessRate は成功率を百分率で返す
func (m *AutoMergeMetrics) GetSuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.TotalAttempts == 0 {
		return 0.0
	}

	return float64(m.SuccessfulMerges) / float64(m.TotalAttempts) * 100.0
}

// GetSuccessRateFormatted はフォーマットされた成功率文字列を返す
func (m *AutoMergeMetrics) GetSuccessRateFormatted() string {
	return fmt.Sprintf("%.2f%%", m.GetSuccessRate())
}

// GetTopFailureReasons は上位N個の失敗理由を取得する
func (m *AutoMergeMetrics) GetTopFailureReasons(limit int) []FailureReason {
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

// Reset はメトリクスをリセットする
func (m *AutoMergeMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalAttempts = 0
	m.SuccessfulMerges = 0
	m.FailedMerges = 0
	m.FailureReasons = make(map[string]int64)
	m.StartTime = time.Now()
	m.LastAttemptTime = time.Time{}
}

// GetUptimeDuration は稼働時間を返す
func (m *AutoMergeMetrics) GetUptimeDuration() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return time.Since(m.StartTime)
}

// GetSnapshot はメトリクスのスナップショットを返す（読み取り専用）
func (m *AutoMergeMetrics) GetSnapshot() AutoMergeMetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// FailureReasonsのコピーを作成
	failureReasons := make(map[string]int64)
	for k, v := range m.FailureReasons {
		failureReasons[k] = v
	}

	return AutoMergeMetricsSnapshot{
		TotalAttempts:    m.TotalAttempts,
		SuccessfulMerges: m.SuccessfulMerges,
		FailedMerges:     m.FailedMerges,
		FailureReasons:   failureReasons,
		StartTime:        m.StartTime,
		LastAttemptTime:  m.LastAttemptTime,
		SuccessRate:      m.getSuccessRateUnsafe(),
		UptimeDuration:   time.Since(m.StartTime),
	}
}

// getSuccessRateUnsafe は内部用の成功率計算（ロックなし）
func (m *AutoMergeMetrics) getSuccessRateUnsafe() float64 {
	if m.TotalAttempts == 0 {
		return 0.0
	}
	return float64(m.SuccessfulMerges) / float64(m.TotalAttempts) * 100.0
}

// AutoMergeMetricsSnapshot はメトリクスの読み取り専用スナップショット
type AutoMergeMetricsSnapshot struct {
	TotalAttempts    int64
	SuccessfulMerges int64
	FailedMerges     int64
	FailureReasons   map[string]int64
	StartTime        time.Time
	LastAttemptTime  time.Time
	SuccessRate      float64
	UptimeDuration   time.Duration
}

// GetSuccessRateFormatted はフォーマットされた成功率文字列を返す
func (s AutoMergeMetricsSnapshot) GetSuccessRateFormatted() string {
	return fmt.Sprintf("%.2f%%", s.SuccessRate)
}

// GetTopFailureReasons は上位N個の失敗理由を取得する
func (s AutoMergeMetricsSnapshot) GetTopFailureReasons(limit int) []FailureReason {
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
