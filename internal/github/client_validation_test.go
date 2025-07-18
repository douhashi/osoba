package github

import (
	"context"
	"testing"
)

func TestClient_ValidationOnly(t *testing.T) {
	ctx := context.Background()

	// バリデーションのみのテスト用クライアント
	client := &GHClient{}

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

	t.Run("TransitionIssueLabel - ownerが空でエラー", func(t *testing.T) {
		_, err := client.TransitionIssueLabel(ctx, "", "repo", 1)
		if err == nil || err.Error() != "owner is required" {
			t.Errorf("expected 'owner is required' error, got %v", err)
		}
	})

	t.Run("TransitionIssueLabel - repoが空でエラー", func(t *testing.T) {
		_, err := client.TransitionIssueLabel(ctx, "owner", "", 1)
		if err == nil || err.Error() != "repo is required" {
			t.Errorf("expected 'repo is required' error, got %v", err)
		}
	})

	t.Run("TransitionIssueLabel - issue番号が0以下でエラー", func(t *testing.T) {
		_, err := client.TransitionIssueLabel(ctx, "owner", "repo", 0)
		if err == nil || err.Error() != "issue number must be positive" {
			t.Errorf("expected 'issue number must be positive' error, got %v", err)
		}
	})

	t.Run("EnsureLabelsExist - ownerが空でエラー", func(t *testing.T) {
		err := client.EnsureLabelsExist(ctx, "", "repo")
		if err == nil || err.Error() != "owner is required" {
			t.Errorf("expected 'owner is required' error, got %v", err)
		}
	})

	t.Run("EnsureLabelsExist - repoが空でエラー", func(t *testing.T) {
		err := client.EnsureLabelsExist(ctx, "owner", "")
		if err == nil || err.Error() != "repo is required" {
			t.Errorf("expected 'repo is required' error, got %v", err)
		}
	})
}
