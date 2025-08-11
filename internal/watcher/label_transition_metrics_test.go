package watcher

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLabelTransitionMetrics(t *testing.T) {
	metrics := NewLabelTransitionMetrics()

	assert.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics.TotalTransitions)
	assert.Equal(t, int64(0), metrics.SuccessfulTransitions)
	assert.Equal(t, int64(0), metrics.FailedTransitions)
	assert.Empty(t, metrics.FailureReasons)
	assert.Empty(t, metrics.TransitionTypes)
	assert.True(t, metrics.StartTime.After(time.Time{}))
}

func TestLabelTransitionMetrics_RecordSuccess(t *testing.T) {
	tests := []struct {
		name       string
		issueNum   int
		transition string
	}{
		{
			name:       "needs-plan to planning",
			issueNum:   123,
			transition: "status:needs-plan->status:planning",
		},
		{
			name:       "ready to implementing",
			issueNum:   456,
			transition: "status:ready->status:implementing",
		},
		{
			name:       "review-requested to reviewing",
			issueNum:   789,
			transition: "status:review-requested->status:reviewing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := NewLabelTransitionMetrics()

			metrics.RecordSuccess(tt.issueNum, tt.transition)

			assert.Equal(t, int64(1), metrics.TotalTransitions)
			assert.Equal(t, int64(1), metrics.SuccessfulTransitions)
			assert.Equal(t, int64(0), metrics.FailedTransitions)
			assert.Equal(t, int64(1), metrics.TransitionTypes[tt.transition])
			assert.True(t, metrics.LastTransitionTime.After(metrics.StartTime))
		})
	}
}

func TestLabelTransitionMetrics_RecordFailure(t *testing.T) {
	tests := []struct {
		name       string
		issueNum   int
		transition string
		reason     string
	}{
		{
			name:       "API error",
			issueNum:   123,
			transition: "status:needs-plan->status:planning",
			reason:     "api_error",
		},
		{
			name:       "permission denied",
			issueNum:   456,
			transition: "status:ready->status:implementing",
			reason:     "permission_denied",
		},
		{
			name:       "timeout",
			issueNum:   789,
			transition: "status:review-requested->status:reviewing",
			reason:     "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := NewLabelTransitionMetrics()

			metrics.RecordFailure(tt.issueNum, tt.transition, tt.reason)

			assert.Equal(t, int64(1), metrics.TotalTransitions)
			assert.Equal(t, int64(0), metrics.SuccessfulTransitions)
			assert.Equal(t, int64(1), metrics.FailedTransitions)
			assert.Equal(t, int64(1), metrics.FailureReasons[tt.reason])
			assert.Equal(t, int64(1), metrics.TransitionTypes[tt.transition])
			assert.True(t, metrics.LastTransitionTime.After(metrics.StartTime))
		})
	}
}

func TestLabelTransitionMetrics_RecordMultipleTransitions(t *testing.T) {
	metrics := NewLabelTransitionMetrics()

	// 複数の成功と失敗を記録
	metrics.RecordSuccess(123, "status:needs-plan->status:planning")
	metrics.RecordSuccess(124, "status:needs-plan->status:planning")
	metrics.RecordFailure(125, "status:ready->status:implementing", "api_error")
	metrics.RecordFailure(126, "status:ready->status:implementing", "api_error")
	metrics.RecordFailure(127, "status:review-requested->status:reviewing", "timeout")

	assert.Equal(t, int64(5), metrics.TotalTransitions)
	assert.Equal(t, int64(2), metrics.SuccessfulTransitions)
	assert.Equal(t, int64(3), metrics.FailedTransitions)
	assert.Equal(t, int64(2), metrics.FailureReasons["api_error"])
	assert.Equal(t, int64(1), metrics.FailureReasons["timeout"])
	assert.Equal(t, int64(2), metrics.TransitionTypes["status:needs-plan->status:planning"])
	assert.Equal(t, int64(2), metrics.TransitionTypes["status:ready->status:implementing"])
	assert.Equal(t, int64(1), metrics.TransitionTypes["status:review-requested->status:reviewing"])
}

func TestLabelTransitionMetrics_GetSuccessRate(t *testing.T) {
	tests := []struct {
		name                  string
		successfulTransitions int64
		totalTransitions      int64
		expectedRate          float64
		expectedFormatted     string
	}{
		{
			name:                  "100% success rate",
			successfulTransitions: 10,
			totalTransitions:      10,
			expectedRate:          100.0,
			expectedFormatted:     "100.00%",
		},
		{
			name:                  "75% success rate",
			successfulTransitions: 3,
			totalTransitions:      4,
			expectedRate:          75.0,
			expectedFormatted:     "75.00%",
		},
		{
			name:                  "0% success rate",
			successfulTransitions: 0,
			totalTransitions:      5,
			expectedRate:          0.0,
			expectedFormatted:     "0.00%",
		},
		{
			name:                  "no transitions",
			successfulTransitions: 0,
			totalTransitions:      0,
			expectedRate:          0.0,
			expectedFormatted:     "0.00%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := &LabelTransitionMetrics{
				TotalTransitions:      tt.totalTransitions,
				SuccessfulTransitions: tt.successfulTransitions,
				FailureReasons:        make(map[string]int64),
				TransitionTypes:       make(map[string]int64),
			}

			assert.Equal(t, tt.expectedRate, metrics.GetSuccessRate())
			assert.Equal(t, tt.expectedFormatted, metrics.GetSuccessRateFormatted())
		})
	}
}

func TestLabelTransitionMetrics_GetTopFailureReasons(t *testing.T) {
	metrics := NewLabelTransitionMetrics()

	// 失敗理由を記録
	metrics.FailureReasons["api_error"] = 10
	metrics.FailureReasons["timeout"] = 5
	metrics.FailureReasons["permission_denied"] = 8
	metrics.FailureReasons["not_found"] = 2

	// 上位3つの失敗理由を取得
	topReasons := metrics.GetTopFailureReasons(3)

	assert.Equal(t, 3, len(topReasons))
	assert.Equal(t, "api_error", topReasons[0].Reason)
	assert.Equal(t, int64(10), topReasons[0].Count)
	assert.Equal(t, "permission_denied", topReasons[1].Reason)
	assert.Equal(t, int64(8), topReasons[1].Count)
	assert.Equal(t, "timeout", topReasons[2].Reason)
	assert.Equal(t, int64(5), topReasons[2].Count)

	// 上限より多く指定した場合
	allReasons := metrics.GetTopFailureReasons(10)
	assert.Equal(t, 4, len(allReasons))

	// 失敗理由がない場合
	emptyMetrics := NewLabelTransitionMetrics()
	emptyReasons := emptyMetrics.GetTopFailureReasons(5)
	assert.Equal(t, 0, len(emptyReasons))
}

func TestLabelTransitionMetrics_GetMostFrequentTransitions(t *testing.T) {
	metrics := NewLabelTransitionMetrics()

	// 遷移パターンを記録
	metrics.TransitionTypes["status:needs-plan->status:planning"] = 15
	metrics.TransitionTypes["status:ready->status:implementing"] = 10
	metrics.TransitionTypes["status:review-requested->status:reviewing"] = 20
	metrics.TransitionTypes["status:requires-changes->status:ready"] = 5

	// 上位3つの遷移パターンを取得
	topTransitions := metrics.GetMostFrequentTransitions(3)

	assert.Equal(t, 3, len(topTransitions))
	assert.Equal(t, "status:review-requested->status:reviewing", topTransitions[0].Type)
	assert.Equal(t, int64(20), topTransitions[0].Count)
	assert.Equal(t, "status:needs-plan->status:planning", topTransitions[1].Type)
	assert.Equal(t, int64(15), topTransitions[1].Count)
	assert.Equal(t, "status:ready->status:implementing", topTransitions[2].Type)
	assert.Equal(t, int64(10), topTransitions[2].Count)
}

func TestLabelTransitionMetrics_Reset(t *testing.T) {
	metrics := NewLabelTransitionMetrics()

	// データを追加
	metrics.RecordSuccess(123, "status:needs-plan->status:planning")
	metrics.RecordFailure(456, "status:ready->status:implementing", "api_error")

	// 初期データが設定されていることを確認
	assert.Equal(t, int64(2), metrics.TotalTransitions)
	assert.NotEmpty(t, metrics.FailureReasons)
	assert.NotEmpty(t, metrics.TransitionTypes)

	// リセット
	metrics.Reset()

	// リセット後の確認
	assert.Equal(t, int64(0), metrics.TotalTransitions)
	assert.Equal(t, int64(0), metrics.SuccessfulTransitions)
	assert.Equal(t, int64(0), metrics.FailedTransitions)
	assert.Empty(t, metrics.FailureReasons)
	assert.Empty(t, metrics.TransitionTypes)
}

func TestLabelTransitionMetrics_GetSnapshot(t *testing.T) {
	metrics := NewLabelTransitionMetrics()

	// データを追加
	metrics.RecordSuccess(123, "status:needs-plan->status:planning")
	metrics.RecordFailure(456, "status:ready->status:implementing", "api_error")

	// スナップショットを取得
	snapshot := metrics.GetSnapshot()

	// スナップショットの内容を確認
	assert.Equal(t, int64(2), snapshot.TotalTransitions)
	assert.Equal(t, int64(1), snapshot.SuccessfulTransitions)
	assert.Equal(t, int64(1), snapshot.FailedTransitions)
	assert.Equal(t, int64(1), snapshot.FailureReasons["api_error"])
	assert.Equal(t, int64(1), snapshot.TransitionTypes["status:needs-plan->status:planning"])
	assert.Equal(t, int64(1), snapshot.TransitionTypes["status:ready->status:implementing"])
	assert.Equal(t, 50.0, snapshot.SuccessRate)

	// スナップショット取得後に元のメトリクスを変更
	metrics.RecordSuccess(789, "status:review-requested->status:reviewing")

	// スナップショットは変更されないことを確認
	assert.Equal(t, int64(2), snapshot.TotalTransitions)
	assert.Equal(t, int64(3), metrics.TotalTransitions)
}

func TestLabelTransitionMetrics_ConcurrentAccess(t *testing.T) {
	metrics := NewLabelTransitionMetrics()

	// 並行アクセスでのdata race検証
	var wg sync.WaitGroup
	concurrency := 100

	// 並行書き込み
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				metrics.RecordSuccess(i, "status:needs-plan->status:planning")
			} else {
				metrics.RecordFailure(i, "status:ready->status:implementing", "api_error")
			}
		}(i)
	}

	// 並行読み込み
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = metrics.GetSuccessRate()
			_ = metrics.GetSnapshot()
			_ = metrics.GetTopFailureReasons(5)
		}()
	}

	wg.Wait()

	// 合計トランザクション数が正しいことを確認
	assert.Equal(t, int64(concurrency), metrics.TotalTransitions)
	assert.Equal(t, int64(concurrency/2), metrics.SuccessfulTransitions)
	assert.Equal(t, int64(concurrency/2), metrics.FailedTransitions)
}

func TestLabelTransitionMetricsSnapshot_Methods(t *testing.T) {
	snapshot := LabelTransitionMetricsSnapshot{
		TotalTransitions:      10,
		SuccessfulTransitions: 7,
		FailedTransitions:     3,
		SuccessRate:           70.0,
		FailureReasons: map[string]int64{
			"api_error":         2,
			"permission_denied": 1,
		},
		TransitionTypes: map[string]int64{
			"status:needs-plan->status:planning":        5,
			"status:ready->status:implementing":         3,
			"status:review-requested->status:reviewing": 2,
		},
	}

	// GetSuccessRateFormatted
	assert.Equal(t, "70.00%", snapshot.GetSuccessRateFormatted())

	// GetTopFailureReasons
	topReasons := snapshot.GetTopFailureReasons(1)
	assert.Equal(t, 1, len(topReasons))
	assert.Equal(t, "api_error", topReasons[0].Reason)

	// GetMostFrequentTransitions
	topTransitions := snapshot.GetMostFrequentTransitions(2)
	assert.Equal(t, 2, len(topTransitions))
	assert.Equal(t, "status:needs-plan->status:planning", topTransitions[0].Type)
	assert.Equal(t, int64(5), topTransitions[0].Count)
}
