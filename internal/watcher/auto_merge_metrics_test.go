package watcher

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAutoMergeMetrics(t *testing.T) {
	metrics := NewAutoMergeMetrics()

	assert.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics.TotalAttempts)
	assert.Equal(t, int64(0), metrics.SuccessfulMerges)
	assert.Equal(t, int64(0), metrics.FailedMerges)
	assert.Empty(t, metrics.FailureReasons)
	assert.True(t, metrics.StartTime.After(time.Time{}))
}

func TestAutoMergeMetrics_RecordSuccess(t *testing.T) {
	metrics := NewAutoMergeMetrics()

	metrics.RecordSuccess(123, 456)

	assert.Equal(t, int64(1), metrics.TotalAttempts)
	assert.Equal(t, int64(1), metrics.SuccessfulMerges)
	assert.Equal(t, int64(0), metrics.FailedMerges)
	assert.True(t, metrics.LastAttemptTime.After(metrics.StartTime))
}

func TestAutoMergeMetrics_RecordFailure(t *testing.T) {
	tests := []struct {
		name   string
		reason string
	}{
		{
			name:   "API error",
			reason: "api_error",
		},
		{
			name:   "not mergeable",
			reason: "not_mergeable",
		},
		{
			name:   "permission denied",
			reason: "permission_denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := NewAutoMergeMetrics()

			metrics.RecordFailure(123, 456, tt.reason)

			assert.Equal(t, int64(1), metrics.TotalAttempts)
			assert.Equal(t, int64(0), metrics.SuccessfulMerges)
			assert.Equal(t, int64(1), metrics.FailedMerges)
			assert.Equal(t, int64(1), metrics.FailureReasons[tt.reason])
			assert.True(t, metrics.LastAttemptTime.After(metrics.StartTime))
		})
	}
}

func TestAutoMergeMetrics_RecordMultipleFailures(t *testing.T) {
	metrics := NewAutoMergeMetrics()

	// 同じ理由で複数回失敗
	metrics.RecordFailure(123, 456, "api_error")
	metrics.RecordFailure(124, 457, "api_error")
	metrics.RecordFailure(125, 458, "not_mergeable")

	assert.Equal(t, int64(3), metrics.TotalAttempts)
	assert.Equal(t, int64(0), metrics.SuccessfulMerges)
	assert.Equal(t, int64(3), metrics.FailedMerges)
	assert.Equal(t, int64(2), metrics.FailureReasons["api_error"])
	assert.Equal(t, int64(1), metrics.FailureReasons["not_mergeable"])
}

func TestAutoMergeMetrics_GetSuccessRate(t *testing.T) {
	tests := []struct {
		name              string
		successfulMerges  int64
		totalAttempts     int64
		expectedRate      float64
		expectedFormatted string
	}{
		{
			name:              "100% success rate",
			successfulMerges:  10,
			totalAttempts:     10,
			expectedRate:      100.0,
			expectedFormatted: "100.00%",
		},
		{
			name:              "75% success rate",
			successfulMerges:  3,
			totalAttempts:     4,
			expectedRate:      75.0,
			expectedFormatted: "75.00%",
		},
		{
			name:              "0% success rate",
			successfulMerges:  0,
			totalAttempts:     5,
			expectedRate:      0.0,
			expectedFormatted: "0.00%",
		},
		{
			name:              "no attempts",
			successfulMerges:  0,
			totalAttempts:     0,
			expectedRate:      0.0,
			expectedFormatted: "0.00%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := &AutoMergeMetrics{
				TotalAttempts:    tt.totalAttempts,
				SuccessfulMerges: tt.successfulMerges,
				FailedMerges:     tt.totalAttempts - tt.successfulMerges,
				FailureReasons:   make(map[string]int64),
				StartTime:        time.Now(),
				LastAttemptTime:  time.Now(),
			}

			rate := metrics.GetSuccessRate()
			formatted := metrics.GetSuccessRateFormatted()

			assert.Equal(t, tt.expectedRate, rate)
			assert.Equal(t, tt.expectedFormatted, formatted)
		})
	}
}

func TestAutoMergeMetrics_GetTopFailureReasons(t *testing.T) {
	metrics := NewAutoMergeMetrics()

	// 異なる失敗理由を記録
	metrics.RecordFailure(123, 456, "api_error")
	metrics.RecordFailure(124, 457, "api_error")
	metrics.RecordFailure(125, 458, "api_error")
	metrics.RecordFailure(126, 459, "not_mergeable")
	metrics.RecordFailure(127, 460, "not_mergeable")
	metrics.RecordFailure(128, 461, "permission_denied")

	topReasons := metrics.GetTopFailureReasons(2)

	assert.Len(t, topReasons, 2)
	assert.Equal(t, "api_error", topReasons[0].Reason)
	assert.Equal(t, int64(3), topReasons[0].Count)
	assert.Equal(t, "not_mergeable", topReasons[1].Reason)
	assert.Equal(t, int64(2), topReasons[1].Count)
}

func TestAutoMergeMetrics_GetTopFailureReasons_LimitExceedsReasons(t *testing.T) {
	metrics := NewAutoMergeMetrics()

	metrics.RecordFailure(123, 456, "api_error")

	topReasons := metrics.GetTopFailureReasons(5) // リクエスト数が実際の理由数より多い場合

	assert.Len(t, topReasons, 1)
	assert.Equal(t, "api_error", topReasons[0].Reason)
	assert.Equal(t, int64(1), topReasons[0].Count)
}

func TestAutoMergeMetrics_GetTopFailureReasons_NoFailures(t *testing.T) {
	metrics := NewAutoMergeMetrics()

	topReasons := metrics.GetTopFailureReasons(3)

	assert.Empty(t, topReasons)
}

func TestAutoMergeMetrics_Reset(t *testing.T) {
	metrics := NewAutoMergeMetrics()

	// メトリクスにデータを追加
	metrics.RecordSuccess(123, 456)
	metrics.RecordFailure(124, 457, "api_error")

	// リセット前の確認
	assert.Equal(t, int64(2), metrics.TotalAttempts)
	assert.Equal(t, int64(1), metrics.SuccessfulMerges)
	assert.Equal(t, int64(1), metrics.FailedMerges)
	assert.Len(t, metrics.FailureReasons, 1)

	originalStartTime := metrics.StartTime
	time.Sleep(1 * time.Millisecond) // 時間差を作る

	metrics.Reset()

	// リセット後の確認
	assert.Equal(t, int64(0), metrics.TotalAttempts)
	assert.Equal(t, int64(0), metrics.SuccessfulMerges)
	assert.Equal(t, int64(0), metrics.FailedMerges)
	assert.Empty(t, metrics.FailureReasons)
	assert.True(t, metrics.StartTime.After(originalStartTime))
	assert.Equal(t, time.Time{}, metrics.LastAttemptTime)
}

func TestAutoMergeMetrics_GetUptimeDuration(t *testing.T) {
	metrics := NewAutoMergeMetrics()

	// 少し時間を待つ
	time.Sleep(1 * time.Millisecond)

	uptime := metrics.GetUptimeDuration()
	assert.True(t, uptime > 0)
}

func TestAutoMergeMetrics_ConcurrentAccess(t *testing.T) {
	metrics := NewAutoMergeMetrics()

	// 並行アクセスのテスト
	done := make(chan bool)

	// goroutine1: 成功を記録
	go func() {
		for i := 0; i < 100; i++ {
			metrics.RecordSuccess(i, i+1000)
		}
		done <- true
	}()

	// goroutine2: 失敗を記録
	go func() {
		for i := 0; i < 100; i++ {
			metrics.RecordFailure(i, i+2000, "api_error")
		}
		done <- true
	}()

	// 両方のgoroutineが完了するまで待つ
	<-done
	<-done

	// 結果確認
	assert.Equal(t, int64(200), metrics.TotalAttempts)
	assert.Equal(t, int64(100), metrics.SuccessfulMerges)
	assert.Equal(t, int64(100), metrics.FailedMerges)
	assert.Equal(t, int64(100), metrics.FailureReasons["api_error"])
}
