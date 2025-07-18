package cmd

import (
	"fmt"

	"github.com/douhashi/osoba/internal/git"
)

// getRepoIdentifier はリポジトリの識別子を取得します
func getRepoIdentifier() (string, error) {
	repoName, err := git.GetRepositoryName()
	if err != nil {
		return "", fmt.Errorf("リポジトリ名の取得に失敗: %w", err)
	}

	// TODO: より正確な方法でオーナーを取得する
	owner := "douhashi"

	return fmt.Sprintf("%s-%s", owner, repoName), nil
}
