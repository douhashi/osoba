package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/douhashi/osoba/internal/logger"
)

// GHClient はghコマンドを使用するGitHub APIクライアント
type GHClient struct {
	logger       logger.Logger
	labelManager LabelManagerInterface
	owner        string
	repo         string
}

// NewClient は新しいGitHub APIクライアントを作成する（ghコマンドベース）
func NewClient(token string) (*GHClient, error) {
	// ghコマンドは環境変数やconfigでトークンを管理するため、ここでは不要
	labelManager := NewGHLabelManager(nil, 3, 1*time.Second)

	client := &GHClient{
		labelManager: labelManager,
	}

	// リポジトリ情報を取得
	client.initRepoInfo(context.Background())

	return client, nil
}

// NewClientWithLogger はログ機能付きの新しいGitHub APIクライアントを作成する（ghコマンドベース）
func NewClientWithLogger(token string, logger logger.Logger) (*GHClient, error) {
	if logger == nil {
		return nil, errors.New("logger is required")
	}

	labelManager := NewGHLabelManager(logger, 3, 1*time.Second)

	client := &GHClient{
		logger:       logger,
		labelManager: labelManager,
	}

	// リポジトリ情報を取得
	client.initRepoInfo(context.Background())

	return client, nil
}

// NewClientWithLabelManager はテスト用のクライアントコンストラクタ
func NewClientWithLabelManager(labelManager LabelManagerInterface) *GHClient {
	return &GHClient{
		labelManager: labelManager,
	}
}

// GetRepository はリポジトリ情報を取得する
func (c *GHClient) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	if owner == "" {
		return nil, errors.New("owner is required")
	}
	if repo == "" {
		return nil, errors.New("repo is required")
	}

	output, err := c.executeGHCommand(ctx, "api", fmt.Sprintf("repos/%s/%s", owner, repo))
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	var repository Repository
	if err := json.Unmarshal(output, &repository); err != nil {
		return nil, fmt.Errorf("failed to parse repository response: %w", err)
	}

	return &repository, nil
}

// ListIssuesByLabels は指定されたラベルのいずれかを持つIssueを取得する（OR条件）
func (c *GHClient) ListIssuesByLabels(ctx context.Context, owner, repo string, labels []string) ([]*Issue, error) {
	if c.logger != nil {
		c.logger.Debug("ListIssuesByLabels called",
			"owner", owner,
			"repo", repo,
			"labels", labels)
	}

	if owner == "" {
		return nil, errors.New("owner is required")
	}
	if repo == "" {
		return nil, errors.New("repo is required")
	}

	// 全てのオープンIssueを取得して、クライアント側でフィルタリング（OR条件）
	args := []string{
		"issue", "list",
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--state", "open",
		"--json", "number,title,labels,state,body,author,assignees,createdAt,updatedAt,closedAt,milestone,comments,url",
		"--limit", "100", // 最大100件まで取得
	}

	output, err := c.executeGHCommand(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	var ghIssues []map[string]interface{}
	if err := json.Unmarshal(output, &ghIssues); err != nil {
		return nil, fmt.Errorf("failed to parse issues response: %w", err)
	}

	issues := make([]*Issue, 0)
	for _, ghIssue := range ghIssues {
		issue, err := convertMapToIssue(ghIssue)
		if err != nil {
			if c.logger != nil {
				c.logger.Warn("Failed to convert issue", "error", err)
			}
			continue
		}

		// ラベルが指定されている場合は、OR条件でフィルタリング
		if len(labels) > 0 {
			hasMatchingLabel := false
			for _, label := range labels {
				if hasLabel(issue, label) {
					hasMatchingLabel = true
					break
				}
			}
			if !hasMatchingLabel {
				continue
			}
		}

		// ghコマンドの出力からHTMLURLを設定
		if issue.HTMLURL == nil && issue.Number != nil {
			url := fmt.Sprintf("https://github.com/%s/%s/issues/%d", owner, repo, *issue.Number)
			issue.HTMLURL = String(url)
		}
		issues = append(issues, issue)
	}

	if c.logger != nil {
		c.logger.Debug("ListIssuesByLabels result",
			"count", len(issues),
			"labels", labels)
	}

	return issues, nil
}

// GetRateLimit はAPI利用制限情報を取得する
func (c *GHClient) GetRateLimit(ctx context.Context) (*RateLimits, error) {
	output, err := c.executeGHCommand(ctx, "api", "rate_limit")
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit: %w", err)
	}

	var response struct {
		Resources struct {
			Core struct {
				Limit     int   `json:"limit"`
				Remaining int   `json:"remaining"`
				Reset     int64 `json:"reset"`
			} `json:"core"`
			Search struct {
				Limit     int   `json:"limit"`
				Remaining int   `json:"remaining"`
				Reset     int64 `json:"reset"`
			} `json:"search"`
		} `json:"resources"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse rate limit response: %w", err)
	}

	// responseからRateLimitsに変換
	rateLimits := &RateLimits{
		Core: &RateLimit{
			Limit:     response.Resources.Core.Limit,
			Remaining: response.Resources.Core.Remaining,
			Reset:     time.Unix(response.Resources.Core.Reset, 0),
		},
		Search: &RateLimit{
			Limit:     response.Resources.Search.Limit,
			Remaining: response.Resources.Search.Remaining,
			Reset:     time.Unix(response.Resources.Search.Reset, 0),
		},
	}

	if c.logger != nil {
		c.logger.Debug("Rate limit info",
			"core_remaining", rateLimits.Core.Remaining,
			"search_remaining", rateLimits.Search.Remaining,
		)
	}

	return rateLimits, nil
}

// TransitionIssueLabel はIssueのラベルを遷移させる
func (c *GHClient) TransitionIssueLabel(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	if owner == "" {
		return false, errors.New("owner is required")
	}
	if repo == "" {
		return false, errors.New("repo is required")
	}
	if issueNumber <= 0 {
		return false, errors.New("issue number must be positive")
	}
	return c.labelManager.TransitionLabelWithRetry(ctx, owner, repo, issueNumber)
}

// TransitionIssueLabelWithInfo はIssueのラベルを遷移させ、詳細情報を返す
func (c *GHClient) TransitionIssueLabelWithInfo(ctx context.Context, owner, repo string, issueNumber int) (bool, *TransitionInfo, error) {
	if owner == "" {
		return false, nil, errors.New("owner is required")
	}
	if repo == "" {
		return false, nil, errors.New("repo is required")
	}
	if issueNumber <= 0 {
		return false, nil, errors.New("issue number must be positive")
	}
	return c.labelManager.TransitionLabelWithInfoWithRetry(ctx, owner, repo, issueNumber)
}

// EnsureLabelsExist は必要なラベルが存在することを確認する
func (c *GHClient) EnsureLabelsExist(ctx context.Context, owner, repo string) error {
	if owner == "" {
		return errors.New("owner is required")
	}
	if repo == "" {
		return errors.New("repo is required")
	}
	return c.labelManager.EnsureLabelsExistWithRetry(ctx, owner, repo)
}

// CreateIssueComment はIssueにコメントを作成する
func (c *GHClient) CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error {
	if owner == "" {
		return errors.New("owner is required")
	}
	if repo == "" {
		return errors.New("repo is required")
	}
	if comment == "" {
		return errors.New("comment is required")
	}

	if _, err := c.executeGHCommand(ctx, "issue", "comment", strconv.Itoa(issueNumber), "--repo", fmt.Sprintf("%s/%s", owner, repo), "--body", comment); err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	if c.logger != nil {
		c.logger.Debug("Created issue comment",
			"owner", owner,
			"repo", repo,
			"issue", issueNumber,
		)
	}

	return nil
}

// RemoveLabel はIssueからラベルを削除する
func (c *GHClient) RemoveLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	if c.logger != nil {
		c.logger.Debug("RemoveLabel called",
			"owner", owner,
			"repo", repo,
			"issue", issueNumber,
			"label", label,
		)
	}

	if owner == "" {
		return errors.New("owner is required")
	}
	if repo == "" {
		return errors.New("repo is required")
	}
	if label == "" {
		return errors.New("label is required")
	}

	if _, err := c.executeGHCommand(ctx, "issue", "edit", fmt.Sprintf("%d", issueNumber), "--repo", fmt.Sprintf("%s/%s", owner, repo), "--remove-label", label); err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to remove label",
				"owner", owner,
				"repo", repo,
				"issue", issueNumber,
				"label", label,
				"error", err,
			)
		}
		return fmt.Errorf("failed to remove label %s from issue #%d: %w", label, issueNumber, err)
	}

	if c.logger != nil {
		c.logger.Debug("Removed label from issue",
			"owner", owner,
			"repo", repo,
			"issue", issueNumber,
			"label", label,
		)
	}

	return nil
}

// AddLabel はIssueにラベルを追加する
func (c *GHClient) AddLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	if c.logger != nil {
		c.logger.Debug("AddLabel called",
			"owner", owner,
			"repo", repo,
			"issue", issueNumber,
			"label", label,
		)
	}

	if owner == "" {
		return errors.New("owner is required")
	}
	if repo == "" {
		return errors.New("repo is required")
	}
	if label == "" {
		return errors.New("label is required")
	}

	if _, err := c.executeGHCommand(ctx, "issue", "edit", fmt.Sprintf("%d", issueNumber), "--repo", fmt.Sprintf("%s/%s", owner, repo), "--add-label", label); err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to add label",
				"owner", owner,
				"repo", repo,
				"issue", issueNumber,
				"label", label,
				"error", err,
			)
		}
		return fmt.Errorf("failed to add label %s to issue #%d: %w", label, issueNumber, err)
	}

	if c.logger != nil {
		c.logger.Debug("Added label to issue",
			"owner", owner,
			"repo", repo,
			"issue", issueNumber,
			"label", label,
		)
	}

	return nil
}

// executeGHCommand はghコマンドを実行する
// ListAllOpenIssues はリポジトリのすべてのオープンなIssueを取得する
func (c *GHClient) ListAllOpenIssues(ctx context.Context, owner, repo string) ([]*Issue, error) {
	if owner == "" {
		return nil, errors.New("owner is required")
	}
	if repo == "" {
		return nil, errors.New("repo is required")
	}

	// ghコマンドを実行してすべてのオープンIssueを取得
	output, err := c.executeGHCommand(ctx, "issue", "list",
		"--repo", owner+"/"+repo,
		"--state", "open", // オープンなIssueのみ
		"--limit", "100", // 最大100件まで取得
		"--json", "number,title,labels,state,body,createdAt,updatedAt,author,url")

	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	var ghIssues []map[string]interface{}
	if err := json.Unmarshal(output, &ghIssues); err != nil {
		return nil, fmt.Errorf("failed to parse issue list: %w", err)
	}

	issues := make([]*Issue, 0, len(ghIssues))
	for _, ghIssue := range ghIssues {
		issue, err := convertMapToIssue(ghIssue)
		if err != nil {
			if c.logger != nil {
				c.logger.Warn("Failed to convert issue", "error", err)
			}
			continue
		}
		issues = append(issues, issue)
	}

	if c.logger != nil {
		c.logger.Debug("Listed all open issues",
			"owner", owner,
			"repo", repo,
			"count", len(issues))
	}

	return issues, nil
}

// ListClosedIssues はリポジトリのクローズされたIssueを取得する
func (c *GHClient) ListClosedIssues(ctx context.Context, owner, repo string) ([]*Issue, error) {
	if owner == "" {
		return nil, errors.New("owner is required")
	}
	if repo == "" {
		return nil, errors.New("repo is required")
	}

	// ghコマンドを実行してクローズされたIssueを取得
	output, err := c.executeGHCommand(ctx, "issue", "list",
		"--repo", owner+"/"+repo,
		"--state", "closed", // クローズされたIssueのみ
		"--limit", "30", // 最大30件まで取得（最近クローズされたもの）
		"--json", "number,title,labels,state,body,createdAt,updatedAt,author,url")
	if err != nil {
		return nil, fmt.Errorf("failed to list closed issues: %w", err)
	}

	var ghIssues []map[string]interface{}
	if err := json.Unmarshal(output, &ghIssues); err != nil {
		return nil, fmt.Errorf("failed to parse closed issue list: %w", err)
	}

	issues := make([]*Issue, 0, len(ghIssues))
	for _, ghIssue := range ghIssues {
		issue, err := convertMapToIssue(ghIssue)
		if err != nil {
			if c.logger != nil {
				c.logger.Warn("Failed to convert closed issue", "error", err)
			}
			continue
		}
		issues = append(issues, issue)
	}

	if c.logger != nil {
		c.logger.Debug("Listed closed issues",
			"owner", owner,
			"repo", repo,
			"count", len(issues))
	}

	return issues, nil
}

// convertMapToIssue はmap[string]interfaceを github.Issue に変換する
func convertMapToIssue(issueMap map[string]interface{}) (*Issue, error) {
	issue := &Issue{}

	// Number
	if numberVal, ok := issueMap["number"]; ok {
		if numberFloat, ok := numberVal.(float64); ok {
			number := int(numberFloat)
			issue.Number = &number
		}
	}

	// Title
	if titleVal, ok := issueMap["title"]; ok {
		if titleStr, ok := titleVal.(string); ok {
			issue.Title = &titleStr
		}
	}

	// State
	if stateVal, ok := issueMap["state"]; ok {
		if stateStr, ok := stateVal.(string); ok {
			state := strings.ToLower(stateStr)
			issue.State = &state
		}
	}

	// HTMLURL
	if urlVal, ok := issueMap["url"]; ok {
		if urlStr, ok := urlVal.(string); ok {
			issue.HTMLURL = &urlStr
		}
	}

	// Body
	if bodyVal, ok := issueMap["body"]; ok {
		if bodyStr, ok := bodyVal.(string); ok {
			issue.Body = &bodyStr
		}
	}

	// User (author)
	if authorVal, ok := issueMap["author"]; ok {
		if authorMap, ok := authorVal.(map[string]interface{}); ok {
			if loginVal, ok := authorMap["login"]; ok {
				if loginStr, ok := loginVal.(string); ok {
					issue.User = &User{Login: &loginStr}
				}
			}
		}
	}

	// Labels
	if labelsVal, ok := issueMap["labels"]; ok {
		if labelsSlice, ok := labelsVal.([]interface{}); ok {
			issue.Labels = make([]*Label, len(labelsSlice))
			for i, labelVal := range labelsSlice {
				if labelMap, ok := labelVal.(map[string]interface{}); ok {
					label := &Label{}
					if nameVal, ok := labelMap["name"]; ok {
						if nameStr, ok := nameVal.(string); ok {
							label.Name = &nameStr
						}
					}
					if descVal, ok := labelMap["description"]; ok {
						if descStr, ok := descVal.(string); ok {
							label.Description = &descStr
						}
					}
					if colorVal, ok := labelMap["color"]; ok {
						if colorStr, ok := colorVal.(string); ok {
							label.Color = &colorStr
						}
					}
					issue.Labels[i] = label
				}
			}
		}
	}

	return issue, nil
}

func (c *GHClient) executeGHCommand(ctx context.Context, args ...string) ([]byte, error) {
	if c.logger != nil {
		c.logger.Debug("Executing gh command",
			"args", args,
		)
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if c.logger != nil {
			c.logger.Error("gh command failed",
				"args", args,
				"error", err,
				"output", string(output),
			)
		}
		// エラーの場合でも、出力がある場合はそれを含める
		if len(output) > 0 {
			return nil, fmt.Errorf("gh command failed: %w, output: %s", err, string(output))
		}
		return nil, fmt.Errorf("gh command failed: %w", err)
	}

	if c.logger != nil {
		c.logger.Debug("gh command completed",
			"args", args,
			"output_size", len(output),
		)
	}

	return output, nil
}

// hasLabel はIssueが指定されたラベルを持っているかを確認する
func hasLabel(issue *Issue, labelName string) bool {
	if issue == nil || issue.Labels == nil {
		return false
	}

	for _, label := range issue.Labels {
		if label.Name != nil && *label.Name == labelName {
			return true
		}
	}
	return false
}

// TransitionLabels は原子的にラベルを削除して追加する
func (c *GHClient) TransitionLabels(ctx context.Context, owner, repo string, issueNumber int, removeLabel, addLabel string) error {
	if c.logger != nil {
		c.logger.Debug("TransitionLabels called",
			"owner", owner,
			"repo", repo,
			"issue", issueNumber,
			"removeLabel", removeLabel,
			"addLabel", addLabel,
		)
	}

	if owner == "" {
		return errors.New("owner is required")
	}
	if repo == "" {
		return errors.New("repo is required")
	}
	if removeLabel == "" {
		return errors.New("removeLabel is required")
	}
	if addLabel == "" {
		return errors.New("addLabel is required")
	}

	// 1つのコマンドで削除と追加を同時に実行（原子的操作）
	if _, err := c.executeGHCommand(ctx, "issue", "edit", fmt.Sprintf("%d", issueNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--remove-label", removeLabel,
		"--add-label", addLabel); err != nil {
		if c.logger != nil {
			c.logger.Error("Failed to transition labels",
				"owner", owner,
				"repo", repo,
				"issue", issueNumber,
				"removeLabel", removeLabel,
				"addLabel", addLabel,
				"error", err,
			)
		}
		return fmt.Errorf("failed to transition labels from %s to %s on issue #%d: %w", removeLabel, addLabel, issueNumber, err)
	}

	if c.logger != nil {
		c.logger.Debug("Successfully transitioned labels",
			"owner", owner,
			"repo", repo,
			"issue", issueNumber,
			"removeLabel", removeLabel,
			"addLabel", addLabel,
		)
	}

	return nil
}

// Ensure GHClient implements GitHubClient interface
var _ GitHubClient = (*GHClient)(nil)
