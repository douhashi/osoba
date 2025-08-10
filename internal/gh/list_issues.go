package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/douhashi/osoba/internal/github"
)

// ListIssuesByLabels は指定されたラベルのいずれかを持つIssueを取得する（OR条件）
func (c *Client) ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*github.Issue, error) {
	if len(labels) == 0 {
		return nil, fmt.Errorf("at least one label is required")
	}

	// 重複を避けるためのマップ
	issueMap := make(map[int]*github.Issue)

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

		// ghIssue から github.Issue に変換し、マップに追加
		for _, ghIssue := range ghIssues {
			issueNumber := ghIssue.Number
			if _, exists := issueMap[issueNumber]; !exists {
				issueMap[issueNumber] = convertToIssue(ghIssue)
			}
		}
	}

	// マップから配列に変換（Issue番号でソート）
	issues := make([]*github.Issue, 0, len(issueMap))
	for _, issue := range issueMap {
		issues = append(issues, issue)
	}

	// Issue番号でソート（安定した順序を保証）
	sort.Slice(issues, func(i, j int) bool {
		return *issues[i].Number < *issues[j].Number
	})

	return issues, nil
}

// ListAllOpenIssues はリポジトリのすべてのオープンなIssueを取得する
func (c *Client) ListAllOpenIssues(ctx context.Context, owner, repo string) ([]*github.Issue, error) {
	// ghコマンドを実行してすべてのオープンIssueを取得
	output, err := c.executor.Execute(ctx, "gh", "issue", "list",
		"--repo", owner+"/"+repo,
		"--state", "open", // オープンなIssueのみ
		"--limit", "100", // 最大100件まで取得
		"--json", "number,title,state,url,body,createdAt,updatedAt,author,labels")
	if err != nil {
		return nil, fmt.Errorf("failed to list open issues: %w", err)
	}

	// JSON出力をパース
	var ghIssues []ghIssue
	if err := json.Unmarshal([]byte(output), &ghIssues); err != nil {
		return nil, fmt.Errorf("failed to parse issues response: %w", err)
	}

	// ghIssue から github.Issue に変換
	results := make([]*github.Issue, 0, len(ghIssues))
	for _, ghIssue := range ghIssues {
		results = append(results, convertToIssue(ghIssue))
	}

	// Issue番号でソート（昇順）
	sort.Slice(results, func(i, j int) bool {
		return *results[i].Number < *results[j].Number
	})

	return results, nil
}

// convertToIssue は ghIssue を github.Issue に変換する
func convertToIssue(ghIssue ghIssue) *github.Issue {
	// ステートを正規化（OPEN -> open, CLOSED -> closed）
	state := strings.ToLower(ghIssue.State)

	issue := &github.Issue{
		Number:    github.Int(ghIssue.Number),
		Title:     github.String(ghIssue.Title),
		State:     github.String(state),
		HTMLURL:   github.String(ghIssue.URL),
		Body:      github.String(ghIssue.Body),
		CreatedAt: &ghIssue.CreatedAt,
		UpdatedAt: &ghIssue.UpdatedAt,
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
