package github

import (
	"context"
	"fmt"
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

// labelTransitioner はghコマンドベースのLabelTransitioner実装
type labelTransitioner struct {
	client GitHubClient
	owner  string
	repo   string
}

// NewLabelTransitioner は新しいLabelTransitionerを作成する（ghコマンドベース）
func NewLabelTransitioner(client GitHubClient, owner, repo string) LabelTransitioner {
	return &labelTransitioner{
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
	// ghクライアントの場合、新しく実装されたAddLabelメソッドを使用
	if err := t.client.AddLabel(ctx, t.owner, t.repo, issueNumber, label); err != nil {
		return fmt.Errorf("add label to issue #%d: %w", issueNumber, err)
	}
	return nil
}

// RemoveLabel はIssueからラベルを削除する
func (t *labelTransitioner) RemoveLabel(ctx context.Context, issueNumber int, label string) error {
	// ghコマンドベースではRemoveLabelForIssueは使用できない

	// ghクライアントの場合、新しく実装されたRemoveLabelメソッドを使用
	if err := t.client.RemoveLabel(ctx, t.owner, t.repo, issueNumber, label); err != nil {
		return fmt.Errorf("remove label from issue #%d: %w", issueNumber, err)
	}
	return nil
}
