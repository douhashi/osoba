package gh

import (
	"context"
	"errors"
	"fmt"
	"os"

	internalGitHub "github.com/douhashi/osoba/internal/github"
)

// Client はghコマンドを使用してGitHub操作を行うクライアント
type Client struct {
	executor CommandExecutor
}

// NewClient は新しいClientを作成する
func NewClient(executor CommandExecutor) (*Client, error) {
	if executor == nil {
		return nil, errors.New("executor is required")
	}
	return &Client{
		executor: executor,
	}, nil
}

// ValidatePrerequisites はghコマンドの前提条件を検証する
func (c *Client) ValidatePrerequisites(ctx context.Context) error {
	// テスト環境では検証をスキップ
	if os.Getenv("OSOBA_TEST_MODE") == "true" {
		return nil
	}

	// ghコマンドがインストールされているか確認
	installed, err := CheckInstalled(ctx, c.executor)
	if err != nil {
		return fmt.Errorf("failed to check gh installation: %w", err)
	}
	if !installed {
		return errors.New("gh command is not installed")
	}

	// ghコマンドが認証済みか確認
	authenticated, err := CheckAuth(ctx, c.executor)
	if err != nil {
		return fmt.Errorf("failed to check gh authentication: %w", err)
	}
	if !authenticated {
		return errors.New("gh command is not authenticated. Run 'gh auth login' first")
	}

	return nil
}

// 以下、GitHubClientインターフェースの実装（スタブ）

// RemoveLabel はIssueからラベルを削除する
func (c *Client) RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	// バリデーション
	if owner == "" {
		return fmt.Errorf("owner is required")
	}
	if repo == "" {
		return fmt.Errorf("repo is required")
	}
	if issueNumber <= 0 {
		return fmt.Errorf("issue number must be positive")
	}
	if label == "" {
		return fmt.Errorf("label is required")
	}

	// 既存のremoveLabelプライベートメソッドを使用
	if err := c.removeLabel(ctx, owner, repo, issueNumber, label); err != nil {
		return fmt.Errorf("failed to remove label %s from issue #%d: %w", label, issueNumber, err)
	}
	return nil
}

// AddLabel はIssueにラベルを追加する
func (c *Client) AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	// バリデーション
	if owner == "" {
		return fmt.Errorf("owner is required")
	}
	if repo == "" {
		return fmt.Errorf("repo is required")
	}
	if issueNumber <= 0 {
		return fmt.Errorf("issue number must be positive")
	}
	if label == "" {
		return fmt.Errorf("label is required")
	}

	// 既存のaddLabelプライベートメソッドを使用
	if err := c.addLabel(ctx, owner, repo, issueNumber, label); err != nil {
		return fmt.Errorf("failed to add label %s to issue #%d: %w", label, issueNumber, err)
	}
	return nil
}

// GitHubClientインターフェースを実装していることをコンパイル時に確認
var _ internalGitHub.GitHubClient = (*Client)(nil)
