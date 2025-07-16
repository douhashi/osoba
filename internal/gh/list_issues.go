package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ListIssuesByLabels は指定されたラベルのいずれかを持つIssueを取得する（OR条件）
func (c *Client) ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]Issue, error) {
	if len(labels) == 0 {
		return nil, fmt.Errorf("at least one label is required")
	}

	// 重複を避けるためのマップ
	issueMap := make(map[int]Issue)

	// 各ラベルについて個別にghコマンドを実行（OR条件を実現）
	for _, label := range labels {
		// ghコマンドを実行
		output, err := c.executor.Execute(ctx, "gh", "issue", "list",
			"--repo", owner+"/"+repo,
			"--label", label,
			"--state", "open", // オープンなIssueのみ
			"--json", "number,title,state,url,body,createdAt,updatedAt,author,labels")
		if err != nil {
			// 1つのラベルでエラーが発生しても続行
			continue
		}

		// JSON出力をパース
		var ghIssues []ghIssue
		if err := json.Unmarshal([]byte(output), &ghIssues); err != nil {
			continue
		}

		// ghIssue から Issue に変換し、マップに追加
		for _, ghIssue := range ghIssues {
			issueNumber := ghIssue.Number
			if _, exists := issueMap[issueNumber]; !exists {
				issueMap[issueNumber] = convertToIssue(ghIssue)
			}
		}
	}

	// マップから配列に変換
	issues := make([]Issue, 0, len(issueMap))
	for _, issue := range issueMap {
		issues = append(issues, issue)
	}

	return issues, nil
}

// convertToIssue は ghIssue を Issue に変換する
func convertToIssue(ghIssue ghIssue) Issue {
	// ステートを正規化（OPEN -> open, CLOSED -> closed）
	state := strings.ToLower(ghIssue.State)

	issue := Issue{
		Number:    ghIssue.Number,
		Title:     ghIssue.Title,
		State:     state,
		URL:       ghIssue.URL,
		Body:      ghIssue.Body,
		CreatedAt: ghIssue.CreatedAt,
		UpdatedAt: ghIssue.UpdatedAt,
		Author: Author{
			Login: ghIssue.Author.Login,
		},
	}

	// ラベルを変換
	if len(ghIssue.Labels) > 0 {
		issue.Labels = make([]Label, len(ghIssue.Labels))
		for i, ghLabel := range ghIssue.Labels {
			issue.Labels[i] = Label{
				Name:        ghLabel.Name,
				Description: ghLabel.Description,
				Color:       ghLabel.Color,
			}
		}
	}

	return issue
}
