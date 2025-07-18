package cmd

import (
	"context"
	"fmt"

	"github.com/douhashi/osoba/internal/utils"
)

// getRepoIdentifier はリポジトリの識別子を取得します
func getRepoIdentifier() (string, error) {
	// GitHubリポジトリ情報を取得
	repoInfo, err := utils.GetGitHubRepoInfo(context.Background())
	if err != nil {
		return "", fmt.Errorf("リポジトリ情報の取得に失敗: %w", err)
	}

	return fmt.Sprintf("%s-%s", repoInfo.Owner, repoInfo.Repo), nil
}
