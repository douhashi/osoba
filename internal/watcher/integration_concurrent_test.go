package watcher

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestConcurrentLabelProcessing は複数のwatcherが同時に動作する場合のラベル処理をテスト
func TestConcurrentLabelProcessing(t *testing.T) {
	log, _ := logger.New(logger.WithLevel("debug"))

	// テスト用のIssue
	issue1 := &github.Issue{
		Number: intPtr(1),
		Title:  stringPtr("Test Issue 1"),
		Labels: []*github.Label{
			{Name: stringPtr("status:needs-plan")},
		},
	}

	issue2 := &github.Issue{
		Number: intPtr(2),
		Title:  stringPtr("Test Issue 2"),
		Labels: []*github.Label{
			{Name: stringPtr("status:ready")},
		},
	}

	// 処理回数をカウント
	var processCount int32
	var labelTransitionCount int32

	// モッククライアントの設定
	mockClient := new(MockGitHubClient)
	mockActionManager := new(MockActionManager)

	// ListIssuesByLabelsのモック
	mockClient.On("ListIssuesByLabels", mock.Anything, "owner", "repo", mock.AnythingOfType("[]string")).
		Return(func(ctx context.Context, owner, repo string, labels []string) []*github.Issue {
			// 最初の呼び出しではトリガーラベル付きのIssueを返す
			count := atomic.LoadInt32(&processCount)
			if count < 2 {
				return []*github.Issue{issue1, issue2}
			}
			// 2回目以降は実行中ラベルが付いたIssueを返す（処理されない）
			return []*github.Issue{
				{
					Number: intPtr(1),
					Title:  stringPtr("Test Issue 1"),
					Labels: []*github.Label{
						{Name: stringPtr("status:needs-plan")},
						{Name: stringPtr("status:planning")}, // 実行中ラベル
					},
				},
				{
					Number: intPtr(2),
					Title:  stringPtr("Test Issue 2"),
					Labels: []*github.Label{
						{Name: stringPtr("status:ready")},
						{Name: stringPtr("status:implementing")}, // 実行中ラベル
					},
				},
			}
		}, nil)

	// アクション実行のモック
	mockActionManager.On("ExecuteAction", mock.Anything, mock.AnythingOfType("*github.Issue")).
		Return(func(ctx context.Context, issue *github.Issue) error {
			atomic.AddInt32(&processCount, 1)
			// 処理に時間がかかることをシミュレート
			time.Sleep(100 * time.Millisecond)
			return nil
		})

	// ラベル操作のモック
	mockClient.On("RemoveLabel", mock.Anything, "owner", "repo", mock.AnythingOfType("int"), mock.AnythingOfType("string")).
		Return(func(ctx context.Context, owner, repo string, issueNumber int, label string) error {
			atomic.AddInt32(&labelTransitionCount, 1)
			return nil
		})

	mockClient.On("AddLabel", mock.Anything, "owner", "repo", mock.AnythingOfType("int"), mock.AnythingOfType("string")).
		Return(nil)

	// 2つのwatcherを作成
	watcher1, err := NewIssueWatcher(mockClient, "owner", "repo", "session1",
		[]string{"status:needs-plan", "status:ready", "status:review-requested"},
		1*time.Second, log) // 最小1秒
	assert.NoError(t, err)
	if err != nil {
		return
	}
	watcher1.actionManager = mockActionManager

	watcher2, err := NewIssueWatcher(mockClient, "owner", "repo", "session2",
		[]string{"status:needs-plan", "status:ready", "status:review-requested"},
		1*time.Second, log) // 最小1秒
	assert.NoError(t, err)
	if err != nil {
		return
	}
	watcher2.actionManager = mockActionManager

	// 並行してwatcherを実行
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		watcher1.Start(ctx, func(issue *github.Issue) {})
	}()

	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		watcher2.Start(ctx, func(issue *github.Issue) {})
	}()

	// watcherの実行を待つ
	wg.Wait()

	// 検証
	// 各Issueは一度だけ処理されるべき（ラベルベースの重複防止が機能）
	assert.Equal(t, int32(2), atomic.LoadInt32(&processCount), "各Issueは一度だけ処理されるべき")

	// ラベル遷移も適切に実行されるべき
	assert.GreaterOrEqual(t, atomic.LoadInt32(&labelTransitionCount), int32(2), "ラベル遷移が実行されるべき")

	// モックの呼び出し検証
	mockClient.AssertExpectations(t)
	mockActionManager.AssertExpectations(t)
}

// TestRapidLabelChanges は高速なラベル変更が発生した場合の動作をテスト
func TestRapidLabelChanges(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.New(logger.WithLevel("debug"))

	var callCount int32

	// モッククライアントの設定
	mockClient := new(MockGitHubClient)
	mockActionManager := new(MockActionManager)

	// 呼び出しごとに異なるラベル状態を返す
	mockClient.On("ListIssuesByLabels", mock.Anything, "owner", "repo", mock.AnythingOfType("[]string")).
		Return(func(ctx context.Context, owner, repo string, labels []string) []*github.Issue {
			count := atomic.AddInt32(&callCount, 1)

			switch count {
			case 1:
				// 最初: needs-planラベル
				return []*github.Issue{{
					Number: intPtr(1),
					Labels: []*github.Label{{Name: stringPtr("status:needs-plan")}},
				}}
			case 2:
				// 次: planningラベル（遷移済み）
				return []*github.Issue{{
					Number: intPtr(1),
					Labels: []*github.Label{
						{Name: stringPtr("status:needs-plan")},
						{Name: stringPtr("status:planning")},
					},
				}}
			case 3:
				// さらに: readyラベルに変更
				return []*github.Issue{{
					Number: intPtr(1),
					Labels: []*github.Label{{Name: stringPtr("status:ready")}},
				}}
			default:
				// 最後: implementingラベル（遷移済み）
				return []*github.Issue{{
					Number: intPtr(1),
					Labels: []*github.Label{
						{Name: stringPtr("status:ready")},
						{Name: stringPtr("status:implementing")},
					},
				}}
			}
		}, nil)

	var executeCount int32
	mockActionManager.On("ExecuteAction", mock.Anything, mock.AnythingOfType("*github.Issue")).
		Return(func(ctx context.Context, issue *github.Issue) error {
			atomic.AddInt32(&executeCount, 1)
			return nil
		})

	// ラベル操作のモック
	var transitionCount int32
	mockClient.On("RemoveLabel", mock.Anything, "owner", "repo", mock.AnythingOfType("int"), mock.AnythingOfType("string")).
		Return(func(ctx context.Context, owner, repo string, issueNumber int, label string) error {
			atomic.AddInt32(&transitionCount, 1)
			return nil
		})

	mockClient.On("AddLabel", mock.Anything, "owner", "repo", mock.AnythingOfType("int"), mock.AnythingOfType("string")).
		Return(nil)

	// watcherを作成
	watcher, err := NewIssueWatcher(mockClient, "owner", "repo", "session1",
		[]string{"status:needs-plan", "status:ready", "status:review-requested"},
		1*time.Second, log) // 最小1秒
	assert.NoError(t, err)
	if err != nil {
		return
	}
	watcher.actionManager = mockActionManager

	// watcherを実行
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	watcher.Start(ctx, func(issue *github.Issue) {})

	// 検証
	// 実行中ラベルがある場合は処理されないので、実行回数は限定的
	assert.LessOrEqual(t, atomic.LoadInt32(&executeCount), int32(2), "実行中ラベルがある場合は処理されない")

	// ラベル遷移は適切に実行される
	assert.GreaterOrEqual(t, atomic.LoadInt32(&transitionCount), int32(2), "ラベル遷移が実行されるべき")
}

// TestNetworkErrorRecovery はネットワークエラー発生時のリカバリをテスト
func TestNetworkErrorRecovery(t *testing.T) {
	ctx := context.Background()
	log, _ := logger.New(logger.WithLevel("debug"))

	var listCallCount int32
	var removeLabelCallCount int32

	// モッククライアントの設定
	mockClient := new(MockGitHubClient)
	mockActionManager := new(MockActionManager)

	// ListIssuesByLabelsで一時的にエラーを返す
	mockClient.On("ListIssuesByLabels", mock.Anything, "owner", "repo", mock.AnythingOfType("[]string")).
		Return(func(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error) {
			count := atomic.AddInt32(&listCallCount, 1)
			if count <= 2 {
				// 最初の2回はエラー
				return nil, assert.AnError
			}
			// 3回目以降は正常
			return []*github.Issue{{
				Number: intPtr(1),
				Labels: []*github.Label{{Name: stringPtr("status:needs-plan")}},
			}}, nil
		})

	mockActionManager.On("ExecuteAction", mock.Anything, mock.AnythingOfType("*github.Issue")).
		Return(nil)

	// RemoveLabelで一時的にエラーを返す（リトライのテスト）
	mockClient.On("RemoveLabel", mock.Anything, "owner", "repo", 1, "status:needs-plan").
		Return(func(ctx context.Context, owner, repo string, issueNumber int, label string) error {
			count := atomic.AddInt32(&removeLabelCallCount, 1)
			if count == 1 {
				// 最初の呼び出しはエラー
				return assert.AnError
			}
			// 2回目以降は成功
			return nil
		})

	mockClient.On("AddLabel", mock.Anything, "owner", "repo", 1, "status:planning").
		Return(nil)

	// watcherを作成
	watcher, err := NewIssueWatcher(mockClient, "owner", "repo", "session1",
		[]string{"status:needs-plan", "status:ready", "status:review-requested"},
		1*time.Second, log) // 最小1秒
	assert.NoError(t, err)
	if err != nil {
		return
	}
	watcher.actionManager = mockActionManager

	// watcherを実行
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	watcher.Start(ctx, func(issue *github.Issue) {})

	// 検証
	// ListIssuesByLabelsは3回以上呼ばれる（エラー後もリトライ）
	assert.GreaterOrEqual(t, atomic.LoadInt32(&listCallCount), int32(3), "エラー後もリトライされるべき")

	// RemoveLabelは2回呼ばれる（リトライメカニズムが動作）
	assert.Equal(t, int32(2), atomic.LoadInt32(&removeLabelCallCount), "リトライメカニズムが動作するべき")

	// モックの呼び出し検証
	mockActionManager.AssertExpectations(t)
}
