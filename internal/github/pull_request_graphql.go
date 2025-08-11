package github

import (
	"context"
	"encoding/json"
	"fmt"
)

// GetPullRequestForIssueViaGraphQL はGraphQL APIを使用してIssueに関連するPRを取得
func (c *GHClient) GetPullRequestForIssueViaGraphQL(ctx context.Context, issueNumber int) (*PullRequest, error) {
	query := fmt.Sprintf(`
	{
		repository(owner: "%s", name: "%s") {
			issue(number: %d) {
				timelineItems(first: 50, itemTypes: [CROSS_REFERENCED_EVENT]) {
					nodes {
						__typename
						... on CrossReferencedEvent {
							source {
								__typename
								... on PullRequest {
									number
									title
									state
									isDraft
									mergeable
									headRefName
									statusCheckRollup {
										state
									}
								}
							}
						}
					}
				}
			}
		}
	}`, c.owner, c.repo, issueNumber)

	args := []string{
		"api", "graphql",
		"-f", fmt.Sprintf("query=%s", query),
	}

	if c.logger != nil {
		c.logger.Debug("Executing GraphQL query for PR search",
			"issue_number", issueNumber,
		)
	}

	output, err := c.executeGHCommand(ctx, args...)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("GraphQL query failed",
				"issue_number", issueNumber,
				"error", err,
			)
		}
		return nil, fmt.Errorf("GraphQL query failed: %w", err)
	}

	// レスポンスをパース
	var response struct {
		Data struct {
			Repository struct {
				Issue struct {
					TimelineItems struct {
						Nodes []struct {
							TypeName string `json:"__typename"`
							Source   struct {
								TypeName          string `json:"__typename"`
								Number            int    `json:"number"`
								Title             string `json:"title"`
								State             string `json:"state"`
								IsDraft           bool   `json:"isDraft"`
								Mergeable         string `json:"mergeable"`
								HeadRefName       string `json:"headRefName"`
								StatusCheckRollup *struct {
									State string `json:"state"`
								} `json:"statusCheckRollup"`
							} `json:"source"`
						} `json:"nodes"`
					} `json:"timelineItems"`
				} `json:"issue"`
			} `json:"repository"`
		} `json:"data"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to parse GraphQL response",
				"issue_number", issueNumber,
				"error", err,
				"raw_output", string(output),
			)
		}
		return nil, fmt.Errorf("failed to parse GraphQL response: %w", err)
	}

	// CrossReferencedEventからPRを探す
	for _, node := range response.Data.Repository.Issue.TimelineItems.Nodes {
		if node.TypeName == "CrossReferencedEvent" && node.Source.TypeName == "PullRequest" {
			// オープンなPRのみを対象にする
			if node.Source.State == "OPEN" {
				checksStatus := ""
				if node.Source.StatusCheckRollup != nil {
					checksStatus = node.Source.StatusCheckRollup.State
				}

				pr := &PullRequest{
					Number:       node.Source.Number,
					Title:        node.Source.Title,
					State:        node.Source.State,
					Mergeable:    node.Source.Mergeable,
					IsDraft:      node.Source.IsDraft,
					HeadRefName:  node.Source.HeadRefName,
					ChecksStatus: checksStatus,
				}

				if c.logger != nil {
					c.logger.Debug("Found PR via GraphQL",
						"issue_number", issueNumber,
						"pr_number", pr.Number,
						"state", pr.State,
						"mergeable", pr.Mergeable,
					)
				}

				return pr, nil
			}
		}
	}

	if c.logger != nil {
		c.logger.Debug("No open PR found via GraphQL",
			"issue_number", issueNumber,
		)
	}

	return nil, nil
}

// ListPullRequestsByLabelsViaGraphQL はGraphQL APIを使用してラベル付きPRを取得
func (c *GHClient) ListPullRequestsByLabelsViaGraphQL(ctx context.Context, owner, repo string, labels []string) ([]*PullRequest, error) {
	// ラベル条件を作成
	labelFilter := ""
	for i, label := range labels {
		if i > 0 {
			labelFilter += " "
		}
		labelFilter += fmt.Sprintf("label:\"%s\"", label)
	}

	query := fmt.Sprintf(`
	{
		repository(owner: "%s", name: "%s") {
			pullRequests(first: 50, states: OPEN, labels: ["%s"]) {
				nodes {
					number
					title
					state
					isDraft
					mergeable
					headRefName
					labels(first: 20) {
						nodes {
							name
						}
					}
					statusCheckRollup {
						state
					}
				}
			}
		}
	}`, owner, repo, labels[0]) // 単一ラベルで検索（複数ラベルのAND条件は別途実装）

	args := []string{
		"api", "graphql",
		"-f", fmt.Sprintf("query=%s", query),
	}

	if c.logger != nil {
		c.logger.Debug("Executing GraphQL query for PR labels",
			"labels", labels,
		)
	}

	output, err := c.executeGHCommand(ctx, args...)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("GraphQL PR labels query failed",
				"labels", labels,
				"error", err,
			)
		}
		return nil, fmt.Errorf("GraphQL PR labels query failed: %w", err)
	}

	// デバッグ用にraw outputをログ出力
	if c.logger != nil {
		c.logger.Debug("GraphQL raw output",
			"output", string(output),
		)
	}

	// レスポンスをパース
	var response struct {
		Data struct {
			Repository struct {
				PullRequests struct {
					Nodes []struct {
						Number      int    `json:"number"`
						Title       string `json:"title"`
						State       string `json:"state"`
						IsDraft     bool   `json:"isDraft"`
						Mergeable   string `json:"mergeable"`
						HeadRefName string `json:"headRefName"`
						Labels      struct {
							Nodes []struct {
								Name string `json:"name"`
							} `json:"nodes"`
						} `json:"labels"`
						StatusCheckRollup *struct {
							State string `json:"state"`
						} `json:"statusCheckRollup"`
					} `json:"nodes"`
				} `json:"pullRequests"`
			} `json:"repository"`
		} `json:"data"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		if c.logger != nil {
			// デバッグ用に詳細な情報を出力
			c.logger.Error("Failed to parse pull request response",
				"error", err,
				"raw_output", string(output),
				"output_length", len(output),
			)
			// 最初の500文字だけ別途出力
			if len(output) > 0 {
				maxLen := 500
				if len(output) < maxLen {
					maxLen = len(output)
				}
				c.logger.Error("Raw output preview",
					"preview", string(output[:maxLen]),
				)
			}
		}
		return nil, fmt.Errorf("failed to parse pull request response (GraphQL): %w", err)
	}

	var prs []*PullRequest

	for _, prNode := range response.Data.Repository.PullRequests.Nodes {
		// PRのラベルを取得
		prLabels := make([]string, 0, len(prNode.Labels.Nodes))
		for _, labelNode := range prNode.Labels.Nodes {
			prLabels = append(prLabels, labelNode.Name)
		}

		// すべての要求ラベルがPRに含まれているかチェック
		hasAllLabels := true
		for _, reqLabel := range labels {
			found := false
			for _, prLabel := range prLabels {
				if prLabel == reqLabel {
					found = true
					break
				}
			}
			if !found {
				hasAllLabels = false
				break
			}
		}

		// すべてのラベルが含まれている場合のみ追加
		if hasAllLabels {
			checksStatus := ""
			if prNode.StatusCheckRollup != nil {
				checksStatus = prNode.StatusCheckRollup.State
			}

			pr := &PullRequest{
				Number:       prNode.Number,
				Title:        prNode.Title,
				State:        prNode.State,
				Mergeable:    prNode.Mergeable,
				IsDraft:      prNode.IsDraft,
				HeadRefName:  prNode.HeadRefName,
				ChecksStatus: checksStatus,
			}
			prs = append(prs, pr)

			if c.logger != nil {
				c.logger.Debug("Found PR with matching labels",
					"pr_number", pr.Number,
					"pr_labels", prLabels,
					"requested_labels", labels,
				)
			}
		}
	}

	if c.logger != nil {
		c.logger.Debug("GraphQL PR labels search completed",
			"labels", labels,
			"found_prs", len(prs),
		)
	}

	return prs, nil
}

// GetClosingIssueNumber はPRに関連付けられたIssue番号を取得
// PRがfixes #123のような形式でIssueをクローズする場合、そのIssue番号を返す
// 関連するIssueがない場合は0を返す
func (c *GHClient) GetClosingIssueNumber(ctx context.Context, prNumber int) (int, error) {
	query := fmt.Sprintf(`
	{
		repository(owner: "%s", name: "%s") {
			pullRequest(number: %d) {
				closingIssuesReferences(first: 1) {
					nodes {
						number
					}
				}
			}
		}
	}`, c.owner, c.repo, prNumber)

	args := []string{
		"api", "graphql",
		"-f", fmt.Sprintf("query=%s", query),
	}

	if c.logger != nil {
		c.logger.Debug("Executing GraphQL query for closing issue",
			"pr_number", prNumber,
		)
	}

	output, err := c.executeGHCommand(ctx, args...)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("GraphQL query for closing issue failed",
				"pr_number", prNumber,
				"error", err,
			)
		}
		return 0, fmt.Errorf("GraphQL query for closing issue failed: %w", err)
	}

	// レスポンスをパース
	var response struct {
		Data struct {
			Repository struct {
				PullRequest *struct {
					ClosingIssuesReferences struct {
						Nodes []struct {
							Number int `json:"number"`
						} `json:"nodes"`
					} `json:"closingIssuesReferences"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to parse closing issue response",
				"pr_number", prNumber,
				"error", err,
				"raw_output", string(output),
			)
		}
		return 0, fmt.Errorf("failed to parse closing issue response: %w", err)
	}

	// PRが存在しない場合
	if response.Data.Repository.PullRequest == nil {
		if c.logger != nil {
			c.logger.Debug("PR not found",
				"pr_number", prNumber,
			)
		}
		return 0, nil
	}

	// closingIssuesReferencesが存在し、最初のIssue番号を取得
	if len(response.Data.Repository.PullRequest.ClosingIssuesReferences.Nodes) > 0 {
		issueNumber := response.Data.Repository.PullRequest.ClosingIssuesReferences.Nodes[0].Number
		if c.logger != nil {
			c.logger.Debug("Found closing issue for PR",
				"pr_number", prNumber,
				"issue_number", issueNumber,
			)
		}
		return issueNumber, nil
	}

	if c.logger != nil {
		c.logger.Debug("No closing issue found for PR",
			"pr_number", prNumber,
		)
	}

	return 0, nil
}
