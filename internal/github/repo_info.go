package github

import (
	"context"
	"encoding/json"
	"strings"
)

// initRepoInfo はリポジトリ情報を初期化する
func (c *GHClient) initRepoInfo(ctx context.Context) error {
	// gh repo view --json owner,name でリポジトリ情報を取得
	output, err := c.executeGHCommand(ctx, "repo", "view", "--json", "owner,name")
	if err != nil {
		// エラーが発生してもフォールバック値を使用
		c.owner = "douhashi"
		c.repo = "osoba"
		return nil
	}

	var repoInfo struct {
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
		Name string `json:"name"`
	}

	if err := json.Unmarshal(output, &repoInfo); err != nil {
		// パースエラーの場合もフォールバック
		c.owner = "douhashi"
		c.repo = "osoba"
		return nil
	}

	c.owner = repoInfo.Owner.Login
	c.repo = repoInfo.Name

	if c.logger != nil {
		c.logger.Debug("Repository info initialized",
			"owner", c.owner,
			"repo", c.repo,
		)
	}

	return nil
}

// GetRepoInfo はowner/repo情報を返す
func (c *GHClient) GetRepoInfo() (string, string) {
	// 未初期化の場合は初期化を試みる
	if c.owner == "" || c.repo == "" {
		c.initRepoInfo(context.Background())
	}

	// それでも空の場合はデフォルト値
	if c.owner == "" {
		c.owner = "douhashi"
	}
	if c.repo == "" {
		c.repo = "osoba"
	}

	return c.owner, c.repo
}

// parseRepoFromURL はGitHub URLからowner/repoを抽出する
func parseRepoFromURL(url string) (string, string) {
	// https://github.com/owner/repo 形式のURLからowner/repoを抽出
	parts := strings.Split(url, "/")
	if len(parts) >= 5 && parts[2] == "github.com" {
		return parts[3], parts[4]
	}
	return "", ""
}
