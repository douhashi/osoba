//go:build integration
// +build integration

package github

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/go-github/v67/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

		// ラベルが作成されたことを確認
		labels, _, err := client.github.Issues.ListLabels(ctx, owner, repo, nil)
		require.NoError(t, err)

		expectedLabels := []string{
			"status:needs-plan",
			"status:planning",
			"status:ready",
			"status:implementing",
			"status:needs-review",
			"status:reviewing",
		}

		labelMap := make(map[string]bool)
		for _, label := range labels {
			labelMap[*label.Name] = true
		}

		for _, expectedLabel := range expectedLabels {
			assert.True(t, labelMap[expectedLabel], "Label %s should exist", expectedLabel)
		}
	})

	// テスト用のIssueを作成してラベル遷移をテスト
	t.Run("Issue作成とラベル遷移のテスト", func(t *testing.T) {
		// テスト用のIssueを作成
		issueRequest := &github.IssueRequest{
			Title:  github.String("Test Issue for Label Transition"),
			Body:   github.String("This is a test issue for label transition"),
			Labels: &[]string{"status:needs-plan"},
		}

		issue, _, err := client.github.Issues.Create(ctx, owner, repo, issueRequest)
		require.NoError(t, err)
		require.NotNil(t, issue)

		issueNumber := *issue.Number
		defer func() {
			// テスト後にIssueをクローズ
			state := "closed"
			_, _, _ = client.github.Issues.Edit(ctx, owner, repo, issueNumber, &github.IssueRequest{
				State: &state,
			})
		}()

		// ラベル遷移を実行
		transitioned, err := client.TransitionIssueLabel(ctx, owner, repo, issueNumber)
		assert.NoError(t, err)
		assert.True(t, transitioned)

		// ラベルが正しく遷移したことを確認
		labels, _, err := client.github.Issues.ListLabelsByIssue(ctx, owner, repo, issueNumber, nil)
		require.NoError(t, err)

		hasPlanning := false
		hasNeedsPlan := false
		for _, label := range labels {
			if *label.Name == "status:planning" {
				hasPlanning = true
			}
			if *label.Name == "status:needs-plan" {
				hasNeedsPlan = true
			}
		}

		assert.True(t, hasPlanning, "Issue should have status:planning label")
		assert.False(t, hasNeedsPlan, "Issue should not have status:needs-plan label")

		// 再度遷移を試みる（既に実行中ラベルがあるのでスキップされるはず）
		transitioned2, err := client.TransitionIssueLabel(ctx, owner, repo, issueNumber)
		assert.NoError(t, err)
		assert.False(t, transitioned2, "Should not transition when already has in-progress label")
	})
}

func TestLabelManagerWithRetryIntegration(t *testing.T) {
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

	// リトライ機能付きラベルマネージャーを作成
	labelManager := NewLabelManagerWithRetry(client.github.Issues, 3, 100*time.Millisecond)

	t.Run("リトライ機能のテスト", func(t *testing.T) {
		// 存在しないリポジトリでテスト（エラーが発生するはず）
		err := labelManager.EnsureLabelsExistWithRetry(ctx, owner, "non-existent-repo-12345")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed after 3 attempts")
	})
}
