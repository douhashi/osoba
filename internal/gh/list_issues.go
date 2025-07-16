package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/go-github/v67/github"
)

// ListIssuesByLabels は指定されたラベルのいずれかを持つIssueを取得する
func (c *Client) ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error) {
	if len(labels) == 0 {
		return nil, fmt.Errorf("at least one label is required")
	}

	// ラベルをカンマ区切りに結合
	labelArg := strings.Join(labels, ",")

	// ghコマンドを実行
	output, err := c.executor.Execute(ctx, "gh", "issue", "list",
		"--repo", owner+"/"+repo,
		"--label", labelArg,
		"--json", "number,title,state,url,body,createdAt,updatedAt,author,labels")
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	// JSON出力をパース
	var ghIssues []ghIssue
	if err := json.Unmarshal([]byte(output), &ghIssues); err != nil {
		return nil, fmt.Errorf("failed to parse issue list: %w", err)
	}

	// ghIssue から github.Issue に変換
	issues := make([]*github.Issue, len(ghIssues))
	for i, ghIssue := range ghIssues {
		issues[i] = convertToGitHubIssue(ghIssue)
	}

	return issues, nil
}

// convertToGitHubIssue は ghIssue を github.Issue に変換する
func convertToGitHubIssue(ghIssue ghIssue) *github.Issue {
	// ステートを正規化（OPEN -> open, CLOSED -> closed）
	state := strings.ToLower(ghIssue.State)

	issue := &github.Issue{
		Number:    github.Int(ghIssue.Number),
		Title:     github.String(ghIssue.Title),
		State:     github.String(state),
		HTMLURL:   github.String(ghIssue.URL),
		Body:      github.String(ghIssue.Body),
		CreatedAt: &github.Timestamp{Time: ghIssue.CreatedAt},
		UpdatedAt: &github.Timestamp{Time: ghIssue.UpdatedAt},
		User: &github.User{
			Login: github.String(ghIssue.Author.Login),
		},
	}

	// ラベルを変換
	if len(ghIssue.Labels) > 0 {
		issue.Labels = make([]*github.Label, len(ghIssue.Labels))
		for i, ghLabel := range ghIssue.Labels {
			issue.Labels[i] = &github.Label{
				Name:        github.String(ghLabel.Name),
				Description: github.String(ghLabel.Description),
				Color:       github.String(ghLabel.Color),
			}
		}
	}

	return issue
}
