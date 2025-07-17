//go:build integration
// +build integration

package github

import (
	"context"
	"os"
	"testing"

	"github.com/douhashi/osoba/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_GitHubClientLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("GITHUB_TOKEN not set, skipping integration test")
	}

	log, err := logger.New(logger.WithLevel("debug"))
	require.NoError(t, err)

	t.Run("実際のGitHub APIリクエストでログが出力される", func(t *testing.T) {
		client, err := NewClientWithLogger(token, log)
		require.NoError(t, err)

		ctx := context.Background()

		// 公開リポジトリの情報を取得
		repo, err := client.GetRepository(ctx, "douhashi", "osoba")
		assert.NoError(t, err)
		assert.NotNil(t, repo)

		// ログが出力されることを視覚的に確認
		t.Logf("Repository: %s", *repo.FullName)
	})

	t.Run("Issue一覧取得でログが出力される", func(t *testing.T) {
		client, err := NewClientWithLogger(token, log)
		require.NoError(t, err)

		ctx := context.Background()

		// Issue一覧を取得（ラベルなし）
		issues, err := client.ListIssuesByLabels(ctx, "golang", "go", []string{})
		assert.NoError(t, err)
		assert.NotEmpty(t, issues)

		// ログが出力されることを視覚的に確認
		t.Logf("Found %d issues", len(issues))
	})

	t.Run("レート制限情報取得でログが出力される", func(t *testing.T) {
		client, err := NewClientWithLogger(token, log)
		require.NoError(t, err)

		ctx := context.Background()

		// レート制限情報を取得
		rateLimit, err := client.GetRateLimit(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, rateLimit)

		// ログが出力されることを視覚的に確認
		t.Logf("Rate limit: %d/%d", rateLimit.Core.Remaining, rateLimit.Core.Limit)
	})
}

func TestLabelManagerIntegration(t *testing.T) {
	// 環境変数からGitHubトークンとテストリポジトリ情報を取得
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("GITHUB_TOKEN is not set")
	}

	owner := os.Getenv("TEST_GITHUB_OWNER")
	if owner == "" {
		owner = "douhashi"
	}

	repo := os.Getenv("TEST_GITHUB_REPO")
	if repo == "" {
		repo = "osoba-test"
	}

	// GitHubクライアントを作成
	client, err := NewClient(token)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("ラベル自動作成と遷移のテスト", func(t *testing.T) {
		// 必要なラベルが存在することを確認
		err := client.EnsureLabelsExist(ctx, owner, repo)
		assert.NoError(t, err)

		// ラベルが作成されたことを確認するため、各ラベルでIssueを検索してみる
		// （ラベルが存在しない場合はエラーになる）
		expectedLabels := []string{
			"status:needs-plan",
			"status:planning",
			"status:ready",
			"status:implementing",
			"status:review-requested",
			"status:reviewing",
		}

		for _, label := range expectedLabels {
			// 各ラベルでIssueを検索（ラベルが存在することの確認）
			_, err := client.ListIssuesByLabels(ctx, owner, repo, []string{label})
			assert.NoError(t, err, "Label %s should exist", label)
		}
	})
}
