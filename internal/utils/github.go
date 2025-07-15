package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// GitHubRepoInfo はGitHubリポジトリの情報を保持する構造体
type GitHubRepoInfo struct {
	Owner string
	Repo  string
}

// ParseGitHubURL はGitHubのURLからowner/repo情報を抽出する
// 以下の形式に対応:
// - https://github.com/owner/repo.git
// - https://github.com/owner/repo
// - git@github.com:owner/repo.git
// - git@github.com:owner/repo
func ParseGitHubURL(url string) (*GitHubRepoInfo, error) {
	// HTTPSのURL形式
	httpsPattern := regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+?)(?:\.git)?$`)
	if matches := httpsPattern.FindStringSubmatch(url); len(matches) == 3 {
		return &GitHubRepoInfo{
			Owner: matches[1],
			Repo:  strings.TrimSuffix(matches[2], ".git"),
		}, nil
	}

	// SSHのURL形式 (git@github.com:owner/repo.git または ssh://git@github.com/owner/repo.git)
	sshPattern := regexp.MustCompile(`^(?:ssh://)?git@github\.com[:/]([^/]+)/([^/]+?)(?:\.git)?$`)
	if matches := sshPattern.FindStringSubmatch(url); len(matches) == 3 {
		return &GitHubRepoInfo{
			Owner: matches[1],
			Repo:  strings.TrimSuffix(matches[2], ".git"),
		}, nil
	}

	return nil, fmt.Errorf("invalid GitHub URL format: %s", url)
}
