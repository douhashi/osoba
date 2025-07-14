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
	"go.uber.org/zap"
)

func TestIntegration_GitHubClientLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("GITHUB_TOKEN not set, skipping integration test")
	}

	zapLogger, err := zap.NewDevelopment()
	require.NoError(t, err)
	log := logger.NewZapLogger(zapLogger)

	t.Run("実際のGitHub APIリクエストでログが出力される", func(t *testing.T) {
		client, err := NewClientWithLogger(token, log)
		require.NoError(t, err)

		ctx := context.Background()

		// 公開リポジトリの情報を取得
		repo, err := client.GetRepository(ctx, "douhashi", "osoba")
		assert.NoError(t, err)
		assert.NotNil(t, repo)

		// ログが出力されることを視覚的に確認
		t.Logf("Repository: %s", repo.GetFullName())
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
