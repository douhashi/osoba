package gh

import (
	"context"
	"encoding/json"
	"fmt"
)

// LabelDefinition defines a GitHub label with its properties
type LabelDefinition struct {
	Name        string
	Color       string
	Description string
}

// 必要なラベル定義
var requiredLabels = []LabelDefinition{
	// Trigger labels
	{
		Name:        "status:needs-plan",
		Color:       "0075ca",
		Description: "Planning phase required",
	},
	{
		Name:        "status:ready",
		Color:       "0e8a16",
		Description: "Ready for implementation",
	},
	{
		Name:        "status:review-requested",
		Color:       "d93f0b",
		Description: "Review requested",
	},
	// In-progress labels
	{
		Name:        "status:planning",
		Color:       "1d76db",
		Description: "Currently in planning phase",
	},
	{
		Name:        "status:implementing",
		Color:       "28a745",
		Description: "Currently being implemented",
	},
	{
		Name:        "status:reviewing",
		Color:       "e99695",
		Description: "Currently under review",
	},
}

// EnsureLabelsExist は必要なラベルがリポジトリに存在することを保証する
func (c *Client) EnsureLabelsExist(ctx context.Context, owner, repo string) error {
	// バリデーション
	if owner == "" {
		return fmt.Errorf("owner is required")
	}
	if repo == "" {
		return fmt.Errorf("repo is required")
	}

	// 既存のラベルを取得
	existingLabels, err := c.getRepositoryLabels(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to list repository labels: %w", err)
	}

	// 既存ラベルのマップを作成
	existingLabelMap := make(map[string]bool)
	for _, label := range existingLabels {
		existingLabelMap[label.Name] = true
	}

	// 不足しているラベルを作成
	for _, requiredLabel := range requiredLabels {
		if !existingLabelMap[requiredLabel.Name] {
			if err := c.createLabel(ctx, owner, repo, requiredLabel); err != nil {
				return fmt.Errorf("failed to create label %s: %w", requiredLabel.Name, err)
			}
		}
	}

	return nil
}

// ghLabelResponse はgh label listの出力を表す構造体
type ghLabelResponse struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// getRepositoryLabels はリポジトリのラベル一覧を取得する
func (c *Client) getRepositoryLabels(ctx context.Context, owner, repo string) ([]ghLabelResponse, error) {
	output, err := c.executor.Execute(ctx, "gh", "label", "list",
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--json", "name,color,description",
		"--limit", "100")
	if err != nil {
		return nil, err
	}

	// JSON出力をパース
	var labels []ghLabelResponse
	if err := json.Unmarshal([]byte(output), &labels); err != nil {
		return nil, fmt.Errorf("failed to parse label list: %w", err)
	}

	return labels, nil
}

// createLabel はリポジトリに新しいラベルを作成する
func (c *Client) createLabel(ctx context.Context, owner, repo string, label LabelDefinition) error {
	_, err := c.executor.Execute(ctx, "gh", "label", "create", label.Name,
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--color", label.Color,
		"--description", label.Description)
	return err
}
