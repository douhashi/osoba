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
								StatusCheckRollup struct {
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
				pr := &PullRequest{
					Number:       node.Source.Number,
					Title:        node.Source.Title,
					State:        node.Source.State,
					Mergeable:    node.Source.Mergeable,
					IsDraft:      node.Source.IsDraft,
					HeadRefName:  node.Source.HeadRefName,
					ChecksStatus: node.Source.StatusCheckRollup.State,
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
