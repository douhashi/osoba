package watcher

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewPRWatcher(t *testing.T) {
	testLogger, _ := logger.New(logger.WithLevel("debug"))

	tests := []struct {
		name          string
		owner         string
		repo          string
		labels        []string
		pollInterval  time.Duration
		expectedError string
	}{
		{
			name:         "valid parameters",
			owner:        "test-owner",
			repo:         "test-repo",
			labels:       []string{"status:lgtm"},
			pollInterval: 20 * time.Second,
		},
		{
			name:          "empty owner",
			owner:         "",
			repo:          "test-repo",
			labels:        []string{"status:lgtm"},
			pollInterval:  20 * time.Second,
			expectedError: "owner is required",
		},
		{
			name:          "empty repo",
			owner:         "test-owner",
			repo:          "",
			labels:        []string{"status:lgtm"},
			pollInterval:  20 * time.Second,
			expectedError: "repo is required",
		},
		{
			name:          "empty labels",
			owner:         "test-owner",
			repo:          "test-repo",
			labels:        []string{},
			pollInterval:  20 * time.Second,
			expectedError: "at least one label is required",
		},
		{
			name:          "invalid poll interval",
			owner:         "test-owner",
			repo:          "test-repo",
			labels:        []string{"status:lgtm"},
			pollInterval:  500 * time.Millisecond,
			expectedError: "poll interval must be at least 1 second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mocks.MockGitHubClient{}

			watcher, err := NewPRWatcher(mockClient, tt.owner, tt.repo, tt.labels, tt.pollInterval, testLogger)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, watcher)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, watcher)
				assert.Equal(t, tt.owner, watcher.owner)
				assert.Equal(t, tt.repo, watcher.repo)
				assert.Equal(t, tt.labels, watcher.labels)
				assert.Equal(t, tt.pollInterval, watcher.GetPollInterval())
			}
		})
	}
}

func TestNewPRWatcherWithConfig(t *testing.T) {
	logger := NewMockLogger()
	cfg := config.NewConfig()
	cfg.GitHub.AutoMergeLGTM = true

	mockClient := &mocks.MockGitHubClient{}

	watcher, err := NewPRWatcherWithConfig(
		mockClient,
		"test-owner",
		"test-repo",
		[]string{"status:lgtm"},
		20*time.Second,
		logger,
		cfg,
		nil, // cleanupManager は nil で内部で作成される
	)

	require.NoError(t, err)
	require.NotNil(t, watcher)
	assert.Equal(t, cfg, watcher.config)
	assert.NotNil(t, watcher.cleanupManager)
	assert.NotNil(t, watcher.autoMergeMetrics)
}

func TestPRWatcher_SetPollInterval(t *testing.T) {
	logger := NewMockLogger()
	mockClient := &mocks.MockGitHubClient{}

	watcher, err := NewPRWatcher(mockClient, "owner", "repo", []string{"status:lgtm"}, 20*time.Second, logger)
	require.NoError(t, err)

	// 有効な間隔を設定
	err = watcher.SetPollInterval(30 * time.Second)
	assert.NoError(t, err)
	assert.Equal(t, 30*time.Second, watcher.GetPollInterval())

	// 無効な間隔を設定
	err = watcher.SetPollInterval(500 * time.Millisecond)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "poll interval must be at least 1 second")
	// 間隔が変更されていないことを確認
	assert.Equal(t, 30*time.Second, watcher.GetPollInterval())
}

func TestPRWatcher_SetPollIntervalForTest(t *testing.T) {
	logger := NewMockLogger()
	mockClient := &mocks.MockGitHubClient{}

	watcher, err := NewPRWatcher(mockClient, "owner", "repo", []string{"status:lgtm"}, 20*time.Second, logger)
	require.NoError(t, err)

	// テスト用メソッドでは1秒未満も許可
	watcher.SetPollIntervalForTest(100 * time.Millisecond)
	assert.Equal(t, 100*time.Millisecond, watcher.GetPollInterval())
}

func TestPRWatcher_GetAutoMergeMetrics(t *testing.T) {
	logger := NewMockLogger()
	mockClient := &mocks.MockGitHubClient{}

	watcher, err := NewPRWatcher(mockClient, "owner", "repo", []string{"status:lgtm"}, 20*time.Second, logger)
	require.NoError(t, err)

	metrics := watcher.GetAutoMergeMetrics()
	assert.Equal(t, int64(0), metrics.TotalAttempts)
	assert.Equal(t, int64(0), metrics.SuccessfulMerges)
	assert.Equal(t, int64(0), metrics.FailedMerges)
	assert.NotNil(t, metrics.FailureReasons)
	assert.Equal(t, float64(0), metrics.SuccessRate)
}

func TestPRWatcher_GetHealthStats(t *testing.T) {
	logger := NewMockLogger()
	mockClient := &mocks.MockGitHubClient{}

	watcher, err := NewPRWatcher(mockClient, "owner", "repo", []string{"status:lgtm"}, 20*time.Second, logger)
	require.NoError(t, err)

	stats := watcher.GetHealthStats()
	assert.Equal(t, 0, stats.TotalExecutions)
	assert.Equal(t, 0, stats.SuccessfulExecutions)
	assert.Equal(t, 0, stats.FailedExecutions)
	assert.True(t, stats.LastExecutionTime.IsZero())
	assert.False(t, stats.StartTime.IsZero())
}

func TestPRWatcher_CheckHealth(t *testing.T) {
	logger := NewMockLogger()
	mockClient := &mocks.MockGitHubClient{}

	watcher, err := NewPRWatcher(mockClient, "owner", "repo", []string{"status:lgtm"}, 20*time.Second, logger)
	require.NoError(t, err)

	// 一度も実行していない状態
	status := watcher.CheckHealth(1 * time.Minute)
	assert.False(t, status.IsHealthy)
	assert.Contains(t, status.Message, "never been executed")

	// 実行時刻を設定
	watcher.mu.Lock()
	watcher.lastExecutionTime = time.Now()
	watcher.totalExecutions = 10
	watcher.successfulExecutions = 10
	watcher.mu.Unlock()

	// 健全な状態
	status = watcher.CheckHealth(1 * time.Minute)
	assert.True(t, status.IsHealthy)
	assert.Contains(t, status.Message, "healthy")

	// 非活性状態（最後の実行から時間が経過）
	watcher.mu.Lock()
	watcher.lastExecutionTime = time.Now().Add(-2 * time.Minute) // 2分前
	watcher.mu.Unlock()

	status = watcher.CheckHealth(1 * time.Minute)
	assert.False(t, status.IsHealthy)
	assert.Contains(t, status.Message, "inactive")

	// 成功率が低い状態
	watcher.mu.Lock()
	watcher.lastExecutionTime = time.Now() // 最新に戻す
	watcher.totalExecutions = 20
	watcher.successfulExecutions = 1 // 5%の成功率
	watcher.mu.Unlock()

	status = watcher.CheckHealth(1 * time.Minute)
	assert.False(t, status.IsHealthy)
	assert.Contains(t, status.Message, "success rate is too low")
}

func TestPRWatcher_checkPRs(t *testing.T) {
	logger := NewMockLogger()
	mockClient := &mocks.MockGitHubClient{}

	// テスト用のPRデータを作成
	testPRs := []*github.PullRequest{
		{
			Number:       123,
			Title:        "Test PR 1",
			State:        "OPEN",
			Mergeable:    "MERGEABLE",
			IsDraft:      false,
			HeadRefName:  "feature-branch",
			ChecksStatus: "SUCCESS",
		},
		{
			Number:       124,
			Title:        "Test PR 2 (Draft)",
			State:        "OPEN",
			Mergeable:    "MERGEABLE",
			IsDraft:      true, // Draftなので処理対象外
			HeadRefName:  "draft-branch",
			ChecksStatus: "SUCCESS",
		},
		{
			Number:       125,
			Title:        "Test PR 3 (Closed)",
			State:        "CLOSED", // Closedなので処理対象外
			Mergeable:    "MERGEABLE",
			IsDraft:      false,
			HeadRefName:  "closed-branch",
			ChecksStatus: "SUCCESS",
		},
	}

	// モックの設定
	mockClient.On("ListPullRequestsByLabels", mock.Anything, "owner", "repo", []string{"status:lgtm"}).
		Return(testPRs, nil)

	watcher, err := NewPRWatcher(mockClient, "owner", "repo", []string{"status:lgtm"}, 20*time.Second, logger)
	require.NoError(t, err)

	// コールバックの呼び出し記録
	var calledPRs []*github.PullRequest
	callback := func(pr *github.PullRequest) {
		calledPRs = append(calledPRs, pr)
	}

	// checkPRsを実行
	watcher.checkPRs(context.Background(), callback)

	// 処理対象のPR（Open && !Draft）のみコールバックが呼ばれることを確認
	require.Len(t, calledPRs, 1)
	assert.Equal(t, 123, calledPRs[0].Number)
	assert.Equal(t, "Test PR 1", calledPRs[0].Title)

	// ヘルスチェック統計が更新されていることを確認
	stats := watcher.GetHealthStats()
	assert.Equal(t, 1, stats.TotalExecutions)
	assert.Equal(t, 1, stats.SuccessfulExecutions)
	assert.Equal(t, 0, stats.FailedExecutions)
	assert.False(t, stats.LastExecutionTime.IsZero())

	mockClient.AssertExpectations(t)
}

func TestPRWatcher_checkPRs_APIError(t *testing.T) {
	logger := NewMockLogger()
	mockClient := &mocks.MockGitHubClient{}

	// API エラーを設定
	mockClient.On("ListPullRequestsByLabels", mock.Anything, "owner", "repo", []string{"status:lgtm"}).
		Return(nil, errors.New("API error"))

	watcher, err := NewPRWatcher(mockClient, "owner", "repo", []string{"status:lgtm"}, 20*time.Second, logger)
	require.NoError(t, err)

	// テスト用の短いポーリング間隔を設定（リトライ間隔を短くするため）
	watcher.SetPollIntervalForTest(100 * time.Millisecond)

	var calledCount int
	callback := func(pr *github.PullRequest) {
		calledCount++
	}

	// checkPRsを実行
	watcher.checkPRs(context.Background(), callback)

	// API エラーの場合、コールバックは呼ばれない
	assert.Equal(t, 0, calledCount)

	// 失敗統計が更新されていることを確認
	stats := watcher.GetHealthStats()
	assert.Equal(t, 1, stats.TotalExecutions)
	assert.Equal(t, 0, stats.SuccessfulExecutions)
	assert.Equal(t, 1, stats.FailedExecutions)

	// リトライが実行されたことを確認（少なくとも1回は呼ばれる）
	mockClient.AssertCalled(t, "ListPullRequestsByLabels", mock.Anything, "owner", "repo", []string{"status:lgtm"})
}

func TestPRWatcher_StartWithAutoMerge(t *testing.T) {
	logger := NewMockLogger()
	mockClient := &mocks.MockGitHubClient{}
	cfg := config.NewConfig()
	cfg.GitHub.AutoMergeLGTM = true

	// テスト用のPRデータ（マージ可能な状態）
	testPR := &github.PullRequest{
		Number:       123,
		Title:        "Test PR",
		State:        "OPEN",
		Mergeable:    "MERGEABLE",
		IsDraft:      false,
		HeadRefName:  "feature-branch",
		ChecksStatus: "SUCCESS",
	}

	// モックの設定
	mockClient.On("ListPullRequestsByLabels", mock.Anything, "owner", "repo", []string{"status:lgtm"}).
		Return([]*github.PullRequest{testPR}, nil).Maybe()

	// GetPullRequestStatus は最新の PR 状態を返す
	mockClient.On("GetPullRequestStatus", mock.Anything, 123).
		Return(testPR, nil).Maybe()

	// MergePullRequest の呼び出し
	mockClient.On("MergePullRequest", mock.Anything, 123).
		Return(nil).Maybe()

	watcher, err := NewPRWatcherWithConfig(mockClient, "owner", "repo", []string{"status:lgtm"}, 20*time.Second, logger, cfg, nil)
	require.NoError(t, err)

	// テスト用の短いポーリング間隔を設定
	watcher.SetPollIntervalForTest(10 * time.Millisecond)

	// タイムアウト付きコンテキスト
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// StartWithAutoMergeを開始（goroutineで実行）
	go watcher.StartWithAutoMerge(ctx)

	// 処理が実行されるまで待機
	time.Sleep(50 * time.Millisecond)

	// モックの呼び出しが行われたことを確認
	// 短い間隔なので複数回呼び出される可能性がある
	mockClient.AssertCalled(t, "ListPullRequestsByLabels", mock.Anything, "owner", "repo", []string{"status:lgtm"})
}

func TestPRWatcher_GetRateLimit(t *testing.T) {
	logger := NewMockLogger()
	mockClient := &mocks.MockGitHubClient{}

	expectedRateLimit := &github.RateLimits{
		Core: &github.RateLimit{
			Limit:     5000,
			Remaining: 4900,
			Reset:     time.Now().Add(1 * time.Hour),
		},
	}

	mockClient.On("GetRateLimit", mock.Anything).Return(expectedRateLimit, nil)

	watcher, err := NewPRWatcher(mockClient, "owner", "repo", []string{"status:lgtm"}, 20*time.Second, logger)
	require.NoError(t, err)

	rateLimit, err := watcher.GetRateLimit(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, expectedRateLimit, rateLimit)

	mockClient.AssertExpectations(t)
}
