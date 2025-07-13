package git

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrNotGitRepository = errors.New("現在のディレクトリはGitリポジトリではありません")
	ErrNoRemoteFound    = errors.New("リモートリポジトリが設定されていません")
)

// GetRepositoryName 現在のディレクトリからGitリポジトリ名を取得
func GetRepositoryName() (string, error) {
	// .gitディレクトリの存在確認
	gitDir := ".git"
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return "", ErrNotGitRepository
	}

	// .git/configファイルを読み込む
	configPath := filepath.Join(gitDir, "config")
	file, err := os.Open(configPath)
	if err != nil {
		return "", fmt.Errorf("git configファイルの読み込みに失敗: %w", err)
	}
	defer file.Close()

	// origin URLを探す
	scanner := bufio.NewScanner(file)
	inRemoteOrigin := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// [remote "origin"]セクションを検出
		if line == `[remote "origin"]` {
			inRemoteOrigin = true
			continue
		}

		// 別のセクションに入ったら終了
		if inRemoteOrigin && strings.HasPrefix(line, "[") {
			break
		}

		// URLを取得
		if inRemoteOrigin && strings.HasPrefix(line, "url =") {
			url := strings.TrimSpace(strings.TrimPrefix(line, "url ="))
			return extractRepoName(url)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("configファイルの読み込みエラー: %w", err)
	}

	return "", ErrNoRemoteFound
}

// extractRepoName URLからリポジトリ名を抽出
func extractRepoName(url string) (string, error) {
	// HTTPSまたはSSH形式のURLから最後の部分を取得
	// https://github.com/user/repo.git
	// git@github.com:user/repo.git

	url = strings.TrimSuffix(url, ".git")

	// 最後のスラッシュまたはコロンの後の部分を取得
	lastSlash := strings.LastIndex(url, "/")
	lastColon := strings.LastIndex(url, ":")

	startIndex := lastSlash
	if lastColon > lastSlash {
		startIndex = lastColon
	}

	if startIndex == -1 || startIndex == len(url)-1 {
		return "", fmt.Errorf("無効なリポジトリURL: %s", url)
	}

	return url[startIndex+1:], nil
}
