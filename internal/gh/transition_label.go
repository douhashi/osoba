package gh

import (
	"context"
	"encoding/json"
	"fmt"

	internalGitHub "github.com/douhashi/osoba/internal/github"
)

// ラベル遷移ルール
var transitionRules = map[string]string{
	"status:needs-plan":       "status:planning",
	"status:ready":            "status:implementing",
	"status:review-requested": "status:reviewing",
}

// 実行中ラベルのセット
var inProgressLabels = map[string]bool{
	"status:planning":     true,
	"status:implementing": true,
	"status:reviewing":    true,
}

// TransitionIssueLabel はIssueのラベルをトリガーラベルから実行中ラベルに遷移させる
func (c *Client) TransitionIssueLabel(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	// バリデーション
	if owner == "" {
		return false, fmt.Errorf("owner is required")
	}
	if repo == "" {
		return false, fmt.Errorf("repo is required")
	}
	if issueNumber <= 0 {
		return false, fmt.Errorf("issue number must be positive")
	}

	// 現在のラベルを取得
	labels, err := c.getIssueLabels(ctx, owner, repo, issueNumber)
	if err != nil {
		return false, fmt.Errorf("failed to get issue labels: %w", err)
	}

	// 既に実行中ラベルがあるかチェック
	for _, label := range labels {
		if inProgressLabels[label] {
			// 既に実行中なのでスキップ
			return false, nil
		}
	}

	// トリガーラベルを探して遷移
	for _, label := range labels {
		if targetLabel, exists := transitionRules[label]; exists {
			// トリガーラベルを削除
			if err := c.removeLabel(ctx, owner, repo, issueNumber, label); err != nil {
				return false, fmt.Errorf("failed to remove label %s: %w", label, err)
			}

			// 実行中ラベルを追加
			if err := c.addLabel(ctx, owner, repo, issueNumber, targetLabel); err != nil {
				// 元のラベルを復元しようとする（ベストエフォート）
				c.addLabel(ctx, owner, repo, issueNumber, label)
				return false, fmt.Errorf("failed to add label %s: %w", targetLabel, err)
			}

			return true, nil
		}
	}

	// トリガーラベルなし
	return false, nil
}

// TransitionIssueLabelWithInfo はIssueのラベルをトリガーラベルから実行中ラベルに遷移させ、遷移情報を返す
func (c *Client) TransitionIssueLabelWithInfo(ctx context.Context, owner, repo string, issueNumber int) (bool, *internalGitHub.TransitionInfo, error) {
	// バリデーション
	if owner == "" {
		return false, nil, fmt.Errorf("owner is required")
	}
	if repo == "" {
		return false, nil, fmt.Errorf("repo is required")
	}
	if issueNumber <= 0 {
		return false, nil, fmt.Errorf("issue number must be positive")
	}

	// 現在のラベルを取得
	labels, err := c.getIssueLabels(ctx, owner, repo, issueNumber)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get issue labels: %w", err)
	}

	// 既に実行中ラベルがあるかチェック
	for _, label := range labels {
		if inProgressLabels[label] {
			// 既に実行中なのでスキップ
			return false, nil, nil
		}
	}

	// トリガーラベルを探して遷移
	for _, label := range labels {
		if targetLabel, exists := transitionRules[label]; exists {
			// トリガーラベルを削除
			if err := c.removeLabel(ctx, owner, repo, issueNumber, label); err != nil {
				return false, nil, fmt.Errorf("failed to remove label %s: %w", label, err)
			}

			// 実行中ラベルを追加
			if err := c.addLabel(ctx, owner, repo, issueNumber, targetLabel); err != nil {
				// 元のラベルを復元しようとする（ベストエフォート）
				c.addLabel(ctx, owner, repo, issueNumber, label)
				return false, nil, fmt.Errorf("failed to add label %s: %w", targetLabel, err)
			}

			// 遷移情報を返す
			info := &internalGitHub.TransitionInfo{
				FromLabel: label,
				ToLabel:   targetLabel,
			}
			return true, info, nil
		}
	}

	// トリガーラベルなし
	return false, nil, nil
}

// getIssueLabels はIssueのラベル一覧を取得する
func (c *Client) getIssueLabels(ctx context.Context, owner, repo string, issueNumber int) ([]string, error) {
	output, err := c.executor.Execute(ctx, "gh", "issue", "view", fmt.Sprintf("%d", issueNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--json", "labels")
	if err != nil {
		return nil, err
	}

	// JSON出力をパース
	var response struct {
		Labels []struct {
			Name  string `json:"name"`
			Color string `json:"color"`
		} `json:"labels"`
	}
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		return nil, fmt.Errorf("failed to parse issue labels: %w", err)
	}

	// ラベル名のリストに変換
	labels := make([]string, len(response.Labels))
	for i, label := range response.Labels {
		labels[i] = label.Name
	}

	return labels, nil
}

// removeLabel はIssueからラベルを削除する
func (c *Client) removeLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	_, err := c.executor.Execute(ctx, "gh", "issue", "edit", fmt.Sprintf("%d", issueNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--remove-label", label)
	return err
}

// addLabel はIssueにラベルを追加する
func (c *Client) addLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	_, err := c.executor.Execute(ctx, "gh", "issue", "edit", fmt.Sprintf("%d", issueNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--add-label", label)
	return err
}
