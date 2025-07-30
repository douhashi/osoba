package cmd

import (
	"context"
	"fmt"
	"os"

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

// getConfigFilePaths は設定ファイルの候補パスを優先順位順に返します
func getConfigFilePaths() []string {
	// カレントディレクトリのみを返す
	return []string{
		".osoba.yml",
		".osoba.yaml",
	}
}

// findConfigFile は実際に存在する設定ファイルのパスを返します
func findConfigFile() (string, bool) {
	paths := getConfigFilePaths()

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}

	return "", false
}
