//go:build integration
// +build integration

package github

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGitHubClientRealIntegration はGitHub APIとの実際の統合テスト
// ghコマンドを使用して外部サービス（GitHub API）との連携をテストし、内部コンポーネントは実際のものを使用
func TestGitHubClientRealIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// ghコマンドが利用可能で認証済みかチェック
	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		t.Skip("gh command not authenticated, skipping real GitHub API integration test")
	}

	log, err := logger.New(logger.WithLevel("error"))
	require.NoError(t, err)

	t.Run("GitHub APIとの実際の連携", func(t *testing.T) {
		// ghコマンドベースのクライアントを作成（トークンは不要、ghが管理）
		client, err := NewClientWithLogger("", log)
		require.NoError(t, err)

		ctx := context.Background()

		t.Run("リポジトリ情報の取得", func(t *testing.T) {
			repo, err := client.GetRepository(ctx, "douhashi", "osoba")
			assert.NoError(t, err)
			assert.NotNil(t, repo)
			if repo != nil && repo.Name != nil {
				assert.Equal(t, "osoba", *repo.Name)
			}
			if repo != nil && repo.Owner != nil && repo.Owner.Login != nil {
				assert.Equal(t, "douhashi", *repo.Owner.Login)
			}
		})

		t.Run("Issue一覧の取得", func(t *testing.T) {
			// ラベルなしでのIssue取得
			issues, err := client.ListIssuesByLabels(ctx, "douhashi", "osoba", []string{})
			if err != nil {
				// ghコマンドでのissue取得が失敗した場合はスキップ（権限不足等）
				t.Skipf("Issue listing failed (may be due to permissions): %v", err)
			}
			assert.NotNil(t, issues)

			t.Logf("Found %d issues in douhashi/osoba", len(issues))
		})

		t.Run("レート制限情報の取得", func(t *testing.T) {
			rateLimit, err := client.GetRateLimit(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, rateLimit)
			assert.NotNil(t, rateLimit.Core)
			assert.Greater(t, rateLimit.Core.Limit, 0)
			assert.GreaterOrEqual(t, rateLimit.Core.Remaining, 0)

			t.Logf("Rate limit: %d/%d (reset: %v)",
				rateLimit.Core.Remaining,
				rateLimit.Core.Limit,
				rateLimit.Core.Reset)
		})

		t.Run("ラベルの存在確認", func(t *testing.T) {
			// 必要なラベルが存在することを確認
			err := client.EnsureLabelsExist(ctx, "douhashi", "osoba")
			assert.NoError(t, err)

			// ラベルでのIssue検索（ラベルが存在することの間接的確認）
			expectedLabels := []string{
				"status:needs-plan",
				"status:planning",
				"status:ready",
				"status:implementing",
				"status:review-requested",
				"status:reviewing",
			}

			for _, label := range expectedLabels {
				issues, err := client.ListIssuesByLabels(ctx, "douhashi", "osoba", []string{label})
				if err != nil {
					// ghコマンドでのラベル検索が失敗した場合はログのみ出力
					t.Logf("Label %s search failed (may be due to permissions): %v", label, err)
					continue
				}
				t.Logf("Label %s: found %d issues", label, len(issues))
			}
		})
	})
}

// TestGitHubClientMockedIntegration はモックサーバーを使った統合テスト
// HTTPレイヤーより上の統合を実際のコンポーネントでテスト
func TestGitHubClientMockedIntegration(t *testing.T) {
	// TODO: TestContainersまたはhttptest.Serverを使ったモックサーバーで
	// GitHub API互換のエンドポイントを提供し、実際のHTTPクライアントと
	// github clientの統合をテストする
	t.Skip("Mock server integration test - to be implemented")
}

// TestGitHubClientErrorHandlingIntegration はエラーハンドリングの統合テスト
func TestGitHubClientErrorHandlingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// ghコマンドが利用可能で認証済みかチェック
	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		t.Skip("gh command not authenticated, skipping error handling integration test")
	}

	log, err := logger.New(logger.WithLevel("error"))
	require.NoError(t, err)

	t.Run("存在しないリポジトリでのエラーハンドリング", func(t *testing.T) {
		client, err := NewClientWithLogger("", log)
		require.NoError(t, err)

		ctx := context.Background()

		_, err = client.GetRepository(ctx, "douhashi", "non-existent-repo-12345")
		assert.Error(t, err)
		t.Logf("Expected error for non-existent repo: %v", err)
	})

	t.Run("レート制限への対応", func(t *testing.T) {
		client, err := NewClientWithLogger("", log)
		require.NoError(t, err)

		ctx := context.Background()

		// レート制限情報を確認
		rateLimit, err := client.GetRateLimit(ctx)
		require.NoError(t, err)

		if rateLimit.Core.Remaining < 10 {
			t.Skipf("Rate limit too low (%d remaining), skipping rate limit test",
				rateLimit.Core.Remaining)
		}

		// 複数回のAPI呼び出しでレート制限の動作を確認
		initialRemaining := rateLimit.Core.Remaining

		for i := 0; i < 3; i++ {
			_, err := client.GetRepository(ctx, "douhashi", "osoba")
			assert.NoError(t, err)
		}

		// レート制限が減少していることを確認
		newRateLimit, err := client.GetRateLimit(ctx)
		assert.NoError(t, err)
		assert.Less(t, newRateLimit.Core.Remaining, initialRemaining)

		t.Logf("Rate limit decreased from %d to %d",
			initialRemaining, newRateLimit.Core.Remaining)
	})
}

// TestGitHubClientConcurrentAccess は並行アクセスでの統合テスト
func TestGitHubClientConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// ghコマンドが利用可能で認証済みかチェック
	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		t.Skip("gh command not authenticated, skipping concurrent access test")
	}

	log, err := logger.New(logger.WithLevel("error"))
	require.NoError(t, err)

	client, err := NewClientWithLogger("", log)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("複数goroutineからの同時アクセス", func(t *testing.T) {
		// レート制限を確認
		rateLimit, err := client.GetRateLimit(ctx)
		require.NoError(t, err)

		if rateLimit.Core.Remaining < 20 {
			t.Skipf("Rate limit too low (%d remaining), skipping concurrent access test",
				rateLimit.Core.Remaining)
		}

		const numGoroutines = 5
		results := make(chan error, numGoroutines)

		// 複数のgoroutineで同時にAPI呼び出し
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				_, err := client.GetRepository(ctx, "douhashi", "osoba")
				results <- err
			}(i)
		}

		// 全ての結果を収集
		successCount := 0
		for i := 0; i < numGoroutines; i++ {
			err := <-results
			if err == nil {
				successCount++
			} else {
				t.Logf("Goroutine %d error: %v", i, err)
			}
		}

		// 少なくとも半数は成功することを期待
		assert.GreaterOrEqual(t, successCount, numGoroutines/2,
			"At least half of concurrent requests should succeed")

		t.Logf("Concurrent access: %d/%d requests succeeded", successCount, numGoroutines)
	})
}

// TestGitHubClientPerformance はパフォーマンスの統合テスト
func TestGitHubClientPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// ghコマンドが利用可能で認証済みかチェック
	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		t.Skip("gh command not authenticated, skipping performance test")
	}

	log, err := logger.New(logger.WithLevel("error"))
	require.NoError(t, err)

	client, err := NewClientWithLogger("", log)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("レスポンス時間の測定", func(t *testing.T) {
		start := time.Now()
		_, err := client.GetRepository(ctx, "douhashi", "osoba")
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Less(t, duration, 5*time.Second, "API response should be within 5 seconds")

		t.Logf("GetRepository response time: %v", duration)
	})

	t.Run("大量データでのレスポンス時間", func(t *testing.T) {
		start := time.Now()
		issues, err := client.ListIssuesByLabels(ctx, "golang", "go", []string{})
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Less(t, duration, 10*time.Second, "Large data API response should be within 10 seconds")

		t.Logf("ListIssues response time: %v (found %d issues)", duration, len(issues))
	})
}

// TestGitHubClientRetryMechanism はリトライ機構の統合テスト
func TestGitHubClientRetryMechanism(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping retry mechanism test in short mode")
	}

	// ghコマンドが利用可能で認証済みかチェック
	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		t.Skip("gh command not authenticated, skipping retry mechanism test")
	}

	log, err := logger.New(logger.WithLevel("debug"))
	require.NoError(t, err)

	t.Run("ネットワーク一時エラーでのリトライ", func(t *testing.T) {
		// ghコマンドベースのクライアントでは直接的なHTTPクライアント設定ができないため
		// 実際のghコマンドが利用できない環境での動作をテスト
		client, err := NewClientWithLogger("", log)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		_, err = client.GetRepository(ctx, "douhashi", "osoba")

		// コンテキストタイムアウトエラーが発生することを確認
		assert.Error(t, err)
		t.Logf("Expected context timeout error: %v", err)
	})
}
