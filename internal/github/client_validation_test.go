package github

import (
	"context"
	"testing"
)

func TestClient_ValidationOnly(t *testing.T) {
	ctx := context.Background()

	// バリデーションのみのテスト用クライアント
	client := &Client{}

	t.Run("GetRepository - ownerが空でエラー", func(t *testing.T) {
		_, err := client.GetRepository(ctx, "", "repo")
		if err == nil || err.Error() != "owner is required" {
			t.Errorf("expected 'owner is required' error, got %v", err)
		}
	})

	t.Run("GetRepository - repoが空でエラー", func(t *testing.T) {
		_, err := client.GetRepository(ctx, "owner", "")
		if err == nil || err.Error() != "repo is required" {
			t.Errorf("expected 'repo is required' error, got %v", err)
		}
	})

	t.Run("ListIssuesByLabels - ownerが空でエラー", func(t *testing.T) {
		_, err := client.ListIssuesByLabels(ctx, "", "repo", []string{"label"})
		if err == nil || err.Error() != "owner is required" {
			t.Errorf("expected 'owner is required' error, got %v", err)
		}
	})

	t.Run("ListIssuesByLabels - repoが空でエラー", func(t *testing.T) {
		_, err := client.ListIssuesByLabels(ctx, "owner", "", []string{"label"})
		if err == nil || err.Error() != "repo is required" {
			t.Errorf("expected 'repo is required' error, got %v", err)
		}
	})
}
