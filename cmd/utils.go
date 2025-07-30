package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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
	var paths []string

	home, err := os.UserHomeDir()
	if err != nil {
		// エラーが発生した場合は空のスライスを返す
		return paths
	}

	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome != "" {
		// XDG_CONFIG_HOMEが設定されている場合
		paths = append(paths,
			filepath.Join(xdgConfigHome, "osoba", "osoba.yml"),
			filepath.Join(xdgConfigHome, "osoba", "osoba.yaml"),
		)
	}

	// デフォルトのパス
	paths = append(paths,
		filepath.Join(home, ".config", "osoba", "osoba.yml"),
		filepath.Join(home, ".config", "osoba", "osoba.yaml"),
		filepath.Join(home, ".osoba.yml"),
		filepath.Join(home, ".osoba.yaml"),
	)

	return paths
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
