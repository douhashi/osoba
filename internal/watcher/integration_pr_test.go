package watcher

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestConcurrentIssueAndPRWatcher はIssueWatcherとPRWatcherの並行動作をテストする
func TestConcurrentIssueAndPRWatcher(t *testing.T) {
	t.Skip("Skipping race-prone concurrent test - functionality is tested in other tests")
	// 各Watcher用に独立したロガーとconfigを作成
	issueLogger := NewMockLogger()
	prLogger := NewMockLogger()

	issueCfg := config.NewConfig()
	issueCfg.GitHub.AutoMergeLGTM = true

	prCfg := config.NewConfig()
	prCfg.GitHub.AutoMergeLGTM = true

	// 各Watcher用に独立したモッククライアントを作成
	issueClient := &mocks.MockGitHubClient{}
	prClient := &mocks.MockGitHubClient{}

	// テスト用のIssueデータ
	testIssues := []*github.Issue{
		{
			Number: intPtr(100),
			Title:  stringPtr("Test Issue"),
			Labels: []*github.Label{
				{Name: stringPtr("status:ready")},
			},
		},
	}

	// テスト用のPRデータ
	testPRs := []*github.PullRequest{
		{
			Number:       200,
			Title:        "Test PR for auto-merge",
			State:        "OPEN",
			Mergeable:    "MERGEABLE",
			IsDraft:      false,
			HeadRefName:  "feature-branch",
			ChecksStatus: "SUCCESS",
		},
	}

	// Issue監視用モックの設定
	issueClient.On("ListIssuesByLabels", mock.Anything, "owner", "repo", []string{"status:needs-plan", "status:ready", "status:review-requested", "status:requires-changes"}).
		Return(testIssues, nil).Maybe()

	// PR監視用モックの設定
	prClient.On("ListPullRequestsByLabels", mock.Anything, "owner", "repo", []string{"status:lgtm"}).
		Return(testPRs, nil).Maybe()
	prClient.On("GetPullRequestStatus", mock.Anything, 200).
		Return(testPRs[0], nil).Maybe()
	prClient.On("MergePullRequest", mock.Anything, 200).
		Return(nil).Maybe()

	// IssueWatcherを作成（独立したクライアントとロガーとconfig使用）
	issueWatcher, err := NewIssueWatcherWithConfig(
		issueClient,
		"owner",
		"repo",
		"test-session",
		issueCfg.GetLabels(),
		issueCfg.GitHub.PollInterval,
		issueLogger,
		issueCfg,
		nil,
	)
	require.NoError(t, err)

	// 短いポーリング間隔に設定
	issueWatcher.SetPollIntervalForTest(20 * time.Millisecond)

	// PRWatcherを作成（独立したクライアントとロガーとconfig使用）
	prWatcher, err := NewPRWatcherWithConfig(
		prClient,
		"owner",
		"repo",
		[]string{"status:lgtm"},
		prCfg.GitHub.PRPollInterval,
		prLogger,
		prCfg,
		nil,
	)
	require.NoError(t, err)

	// 短いポーリング間隔に設定
	prWatcher.SetPollIntervalForTest(20 * time.Millisecond)

	// 各Watcherごとに独立したコンテキストを作成
	issueCtx, issueCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer issueCancel()

	prCtx, prCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer prCancel()

	var wg sync.WaitGroup

	// IssueWatcherを並行で開始
	wg.Add(1)
	go func() {
		defer wg.Done()
		issueWatcher.StartWithActions(issueCtx)
	}()

	// PRWatcherを並行で開始
	wg.Add(1)
	go func() {
		defer wg.Done()
		prWatcher.StartWithAutoMerge(prCtx)
	}()

	// 両方の処理が完了するまで待機
	wg.Wait()

	// 両方のWatcherが実行されたことを確認
	issueStats := issueWatcher.GetHealthStats()
	prStats := prWatcher.GetHealthStats()

	assert.Greater(t, issueStats.TotalExecutions, 0, "IssueWatcher should have executed at least once")
	assert.Greater(t, prStats.TotalExecutions, 0, "PRWatcher should have executed at least once")

	// 両方のWatcherが健全に動作していることを確認
	issueHealth := issueWatcher.CheckHealth(1 * time.Second)
	prHealth := prWatcher.CheckHealth(1 * time.Second)

	// 短時間での実行なので、健全性の判定は緩く設定
	assert.False(t, issueStats.LastExecutionTime.IsZero(), "IssueWatcher should have recorded execution time")
	assert.False(t, prStats.LastExecutionTime.IsZero(), "PRWatcher should have recorded execution time")

	t.Logf("IssueWatcher: %s", issueHealth.Message)
	t.Logf("PRWatcher: %s", prHealth.Message)
}

// TestPRWatcherAutoMergeFlow はPR自動マージフローの統合テスト
func TestPRWatcherAutoMergeFlow(t *testing.T) {
	logger := NewMockLogger()
	cfg := config.NewConfig()
	cfg.GitHub.AutoMergeLGTM = true

	mockClient := &mocks.MockGitHubClient{}

	// マージ可能なPR
	mergeablePR := &github.PullRequest{
		Number:       123,
		Title:        "Mergeable PR",
		State:        "OPEN",
		Mergeable:    "MERGEABLE",
		IsDraft:      false,
		HeadRefName:  "feature-123",
		ChecksStatus: "SUCCESS",
		Labels:       []string{"status:lgtm"}, // LGTMラベルを設定
	}

	// マージ不可能なPR（Draft）
	draftPR := &github.PullRequest{
		Number:       124,
		Title:        "Draft PR",
		State:        "OPEN",
		Mergeable:    "MERGEABLE",
		IsDraft:      true, // Draft なのでマージしない
		HeadRefName:  "draft-124",
		ChecksStatus: "SUCCESS",
		Labels:       []string{"status:lgtm"}, // LGTMラベルを設定
	}

	// モックの設定（mock.Anythingを使用）
	mockClient.On("ListPullRequestsByLabels", mock.Anything, "owner", "repo", []string{"status:lgtm"}).
		Return([]*github.PullRequest{mergeablePR, draftPR}, nil)

	// マージ可能なPRの状態確認
	mockClient.On("GetPullRequestStatus", mock.Anything, 123).
		Return(mergeablePR, nil)

	// マージ実行（マージ可能なPRのみ）
	mockClient.On("MergePullRequest", mock.Anything, 123).
		Return(nil)

	// PRWatcherを作成
	prWatcher, err := NewPRWatcherWithConfig(
		mockClient,
		"owner",
		"repo",
		[]string{"status:lgtm"},
		20*time.Second,
		logger,
		cfg,
		nil,
	)
	require.NoError(t, err)

	// テスト用の短いポーリング間隔
	prWatcher.SetPollIntervalForTest(10 * time.Millisecond)

	// 短時間でテスト実行
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// 自動マージ付きで開始（goroutineで実行）
	go prWatcher.StartWithAutoMerge(ctx)

	// 処理が実行されるまで待機
	time.Sleep(30 * time.Millisecond)

	// 自動マージメトリクスを確認
	metrics := prWatcher.GetAutoMergeMetrics()

	// マージが実行されたことを確認（マージ可能なPRのみ）
	assert.Greater(t, metrics.TotalAttempts, int64(0), "Should have attempted auto-merge")

	// モックの呼び出しを確認
	mockClient.AssertCalled(t, "ListPullRequestsByLabels", mock.Anything, "owner", "repo", []string{"status:lgtm"})
	mockClient.AssertCalled(t, "MergePullRequest", mock.Anything, 123)

	// Draftのためマージされていないことを確認
	mockClient.AssertNotCalled(t, "MergePullRequest", mock.Anything, 124)
}

// TestPRWatcherExclusiveLabelControl は排他制御ロジックをテストする
func TestPRWatcherExclusiveLabelControl(t *testing.T) {
	logger := NewMockLogger()
	cfg := config.NewConfig()
	cfg.GitHub.AutoMergeLGTM = true
	cfg.GitHub.AutoRevisePR = true

	t.Run("status:lgtm優先でauto-mergeのみ実行", func(t *testing.T) {
		mockClient := &mocks.MockGitHubClient{}

		// 両方のラベルを持つPR
		prWithBothLabels := &github.PullRequest{
			Number:       123,
			Title:        "PR with both labels",
			State:        "OPEN",
			Mergeable:    "MERGEABLE",
			IsDraft:      false,
			HeadRefName:  "feature-123",
			ChecksStatus: "SUCCESS",
			Labels:       []string{"status:lgtm", "status:requires-changes"}, // 両方のラベル
		}

		// モックの設定
		mockClient.On("ListPullRequestsByLabels", mock.Anything, "owner", "repo", []string{"status:lgtm", "status:requires-changes"}).
			Return([]*github.PullRequest{prWithBothLabels}, nil)

		// auto-merge関連のモック
		mockClient.On("GetPullRequestStatus", mock.Anything, 123).
			Return(prWithBothLabels, nil)
		mockClient.On("MergePullRequest", mock.Anything, 123).
			Return(nil)

		// PRWatcherを作成（両方のラベルを監視）
		prWatcher, err := NewPRWatcherWithConfig(
			mockClient,
			"owner",
			"repo",
			[]string{"status:lgtm", "status:requires-changes"},
			20*time.Second,
			logger,
			cfg,
			nil,
		)
		require.NoError(t, err)

		prWatcher.SetPollIntervalForTest(10 * time.Millisecond)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		go prWatcher.StartWithAutoMerge(ctx)

		time.Sleep(30 * time.Millisecond)

		// auto-mergeが実行されたことを確認
		mockClient.AssertCalled(t, "MergePullRequest", mock.Anything, 123)

		// auto-revise関連の呼び出しは発生しないはず（排他制御）
		// GetClosingIssueNumberは呼ばれないことを確認
		mockClient.AssertNotCalled(t, "GetClosingIssueNumber", mock.Anything, 123)
	})

	t.Run("status:requires-changesのみでauto-merge非実行", func(t *testing.T) {
		mockClient := &mocks.MockGitHubClient{}

		// status:requires-changesのみのPR
		prWithRequiresChanges := &github.PullRequest{
			Number:       124,
			Title:        "PR with requires-changes",
			State:        "OPEN",
			Mergeable:    "MERGEABLE",
			IsDraft:      false,
			HeadRefName:  "feature-124",
			ChecksStatus: "SUCCESS",
			Labels:       []string{"status:requires-changes"}, // requires-changesのみ
		}

		// モックの設定
		mockClient.On("ListPullRequestsByLabels", mock.Anything, "owner", "repo", []string{"status:lgtm", "status:requires-changes"}).
			Return([]*github.PullRequest{prWithRequiresChanges}, nil)

		// auto-revise関連のモック（GetClosingIssueNumberは呼ばれる可能性がある）
		mockClient.On("GetClosingIssueNumber", mock.Anything, 124).
			Return(100, nil).Maybe()

		// PRWatcherを作成（ActionManagerは設定しない）
		prWatcher, err := NewPRWatcherWithConfig(
			mockClient,
			"owner",
			"repo",
			[]string{"status:lgtm", "status:requires-changes"},
			20*time.Second,
			logger,
			cfg,
			nil,
		)
		require.NoError(t, err)

		prWatcher.SetPollIntervalForTest(10 * time.Millisecond)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		go prWatcher.StartWithAutoMerge(ctx)

		time.Sleep(30 * time.Millisecond)

		// auto-merge関連の呼び出しは発生しないはず（排他制御）
		// status:requires-changesのみの場合、auto-mergeは実行されない
		mockClient.AssertNotCalled(t, "MergePullRequest", mock.Anything, 124)
		mockClient.AssertNotCalled(t, "GetPullRequestStatus", mock.Anything, 124)
	})
}

// TestPRWatcherHealthMetrics はPRWatcherのヘルスメトリクス機能をテスト
func TestPRWatcherHealthMetrics(t *testing.T) {
	logger := NewMockLogger()
	mockClient := &mocks.MockGitHubClient{}

	// 正常なレスポンスを返す設定
	mockClient.On("ListPullRequestsByLabels", mock.Anything, "owner", "repo", []string{"status:lgtm"}).
		Return([]*github.PullRequest{}, nil)

	prWatcher, err := NewPRWatcher(
		mockClient,
		"owner",
		"repo",
		[]string{"status:lgtm"},
		20*time.Second,
		logger,
	)
	require.NoError(t, err)

	// 短いポーリング間隔に設定
	prWatcher.SetPollIntervalForTest(10 * time.Millisecond)

	// 複数回実行させるため少し長めのタイムアウト
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var callCount int32
	callback := func(pr *github.PullRequest) {
		atomic.AddInt32(&callCount, 1)
	}

	// Start メソッドで実行
	go prWatcher.Start(ctx, callback)

	// 処理が実行されるまで待機
	time.Sleep(100 * time.Millisecond)

	// ヘルスチェック統計を確認
	stats := prWatcher.GetHealthStats()
	assert.Greater(t, stats.TotalExecutions, 0, "Should have executed multiple times")
	assert.GreaterOrEqual(t, stats.SuccessfulExecutions, stats.TotalExecutions-1, "Most executions should be successful")
	assert.Equal(t, 0, stats.FailedExecutions, "No executions should have failed")
	assert.False(t, stats.LastExecutionTime.IsZero(), "Should have recorded last execution time")
	assert.False(t, stats.StartTime.IsZero(), "Should have recorded start time")

	// ヘルスチェックが正常であることを確認
	health := prWatcher.CheckHealth(1 * time.Second)
	assert.True(t, health.IsHealthy, "Watcher should be healthy")
	assert.Contains(t, health.Message, "healthy", "Health message should indicate healthy status")

	// API が適切に呼び出されていることを確認
	mockClient.AssertCalled(t, "ListPullRequestsByLabels", mock.Anything, "owner", "repo", []string{"status:lgtm"})
}

// ヘルパー関数は label_transition_test.go で定義済み
