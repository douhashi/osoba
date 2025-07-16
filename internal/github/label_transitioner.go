package github

import (
	"context"
	"fmt"
	"net/http"
)

// LabelTransitioner はフェーズ固有のラベル遷移を行うインターフェース
type LabelTransitioner interface {
	// TransitionLabel はIssueのラベルを明示的に遷移させる（fromラベルを削除し、toラベルを追加）
	TransitionLabel(ctx context.Context, issueNumber int, from, to string) error
	// AddLabel はIssueにラベルを追加する
	AddLabel(ctx context.Context, issueNumber int, label string) error
	// RemoveLabel はIssueからラベルを削除する
	RemoveLabel(ctx context.Context, issueNumber int, label string) error
}

// LabelTransitionerService はGitHub APIのラベル操作インターフェース
type LabelTransitionerService interface {
	AddLabelsToIssue(ctx context.Context, owner, repo string, number int, labels []string) ([]*Label, *Response, error)
	RemoveLabelForIssue(ctx context.Context, owner, repo string, number int, label string) (*Response, error)
}

// labelTransitioner はLabelTransitionerの実装
type labelTransitioner struct {
	client LabelTransitionerService
	owner  string
	repo   string
}

// NewLabelTransitioner は新しいLabelTransitionerを作成する
func NewLabelTransitioner(client LabelTransitionerService, owner, repo string) LabelTransitioner {
	return &labelTransitioner{
		client: client,
		owner:  owner,
		repo:   repo,
	}
}

// gitHubClientLabelTransitioner はGitHubClientインターフェースを使用するLabelTransitioner実装
type gitHubClientLabelTransitioner struct {
	client GitHubClient
	owner  string
	repo   string
}

// NewLabelTransitionerFromGitHubClient はGitHubClientインターフェースを使用するLabelTransitionerを作成する
func NewLabelTransitionerFromGitHubClient(client GitHubClient, owner, repo string) LabelTransitioner {
	return &gitHubClientLabelTransitioner{
		client: client,
		owner:  owner,
		repo:   repo,
	}
}

// TransitionLabel はIssueのラベルを遷移させる
func (t *labelTransitioner) TransitionLabel(ctx context.Context, issueNumber int, from, to string) error {
	// まず既存のラベルを削除
	if err := t.RemoveLabel(ctx, issueNumber, from); err != nil {
		return fmt.Errorf("remove label %s: %w", from, err)
	}

	// 新しいラベルを追加
	if err := t.AddLabel(ctx, issueNumber, to); err != nil {
		return fmt.Errorf("add label %s: %w", to, err)
	}

	return nil
}

// AddLabel はIssueにラベルを追加する
func (t *labelTransitioner) AddLabel(ctx context.Context, issueNumber int, label string) error {
	_, _, err := t.client.AddLabelsToIssue(ctx, t.owner, t.repo, issueNumber, []string{label})
	if err != nil {
		return fmt.Errorf("add label to issue #%d: %w", issueNumber, err)
	}
	return nil
}

// RemoveLabel はIssueからラベルを削除する
func (t *labelTransitioner) RemoveLabel(ctx context.Context, issueNumber int, label string) error {
	_, err := t.client.RemoveLabelForIssue(ctx, t.owner, t.repo, issueNumber, label)
	if err != nil {
		return fmt.Errorf("remove label from issue #%d: %w", issueNumber, err)
	}
	return nil
}

// gitHubClientLabelTransitionerのメソッド実装

// TransitionLabel はIssueのラベルを遷移させる
func (t *gitHubClientLabelTransitioner) TransitionLabel(ctx context.Context, issueNumber int, from, to string) error {
	// まず既存のラベルを削除
	if err := t.RemoveLabel(ctx, issueNumber, from); err != nil {
		return fmt.Errorf("remove label %s: %w", from, err)
	}

	// 新しいラベルを追加
	if err := t.AddLabel(ctx, issueNumber, to); err != nil {
		return fmt.Errorf("add label %s: %w", to, err)
	}

	return nil
}

// AddLabel はIssueにラベルを追加する
func (t *gitHubClientLabelTransitioner) AddLabel(ctx context.Context, issueNumber int, label string) error {
	// GitHub APIの場合
	if apiClient, ok := t.client.(*Client); ok {
		_, _, err := apiClient.github.Issues.AddLabelsToIssue(ctx, t.owner, t.repo, issueNumber, []string{label})
		if err != nil {
			return fmt.Errorf("add label to issue #%d: %w", issueNumber, err)
		}
		return nil
	}

	// ghクライアントの場合、新しく実装されたAddLabelメソッドを使用
	if err := t.client.AddLabel(ctx, t.owner, t.repo, issueNumber, label); err != nil {
		return fmt.Errorf("add label to issue #%d: %w", issueNumber, err)
	}
	return nil
}

// RemoveLabel はIssueからラベルを削除する
func (t *gitHubClientLabelTransitioner) RemoveLabel(ctx context.Context, issueNumber int, label string) error {
	// GitHub APIの場合
	if apiClient, ok := t.client.(*Client); ok {
		_, err := apiClient.github.Issues.RemoveLabelForIssue(ctx, t.owner, t.repo, issueNumber, label)
		if err != nil {
			return fmt.Errorf("remove label from issue #%d: %w", issueNumber, err)
		}
		return nil
	}

	// ghクライアントの場合、新しく実装されたRemoveLabelメソッドを使用
	if err := t.client.RemoveLabel(ctx, t.owner, t.repo, issueNumber, label); err != nil {
		return fmt.Errorf("remove label from issue #%d: %w", issueNumber, err)
	}
	return nil
}
