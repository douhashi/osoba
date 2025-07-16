package gh

import (
	"context"
	"errors"
	"fmt"

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

// EnsureLabelsExist は必要なラベルがリポジトリに存在することを保証する
func (c *Client) EnsureLabelsExist(ctx context.Context, owner, repo string) error {
	// TODO: 実装
	return fmt.Errorf("not implemented")
}

// GitHubClientインターフェースを実装していることをコンパイル時に確認
var _ internalGitHub.GitHubClient = (*Client)(nil)
