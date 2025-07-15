package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/logger"
)

// GetGitHubRepoInfoError は詳細なエラー情報を持つエラー型
type GetGitHubRepoInfoError struct {
	Step    string // どの段階で失敗したか
	Cause   error  // 根本的な原因
	Message string // ユーザー向けメッセージ
}

func (e *GetGitHubRepoInfoError) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *GetGitHubRepoInfoError) Unwrap() error {
	return e.Cause
}

// GetGitHubRepoInfo は現在のGitリポジトリからGitHubリポジトリ情報を取得する
// 各コマンドで統一的に使用される関数
func GetGitHubRepoInfo(ctx context.Context) (*GitHubRepoInfo, error) {
	// ロガーを作成（エラーが発生してもログは出力しない）
	log, err := logger.New()
	if err != nil {
		// ロガー作成に失敗した場合は、軽量なリポジトリインスタンスを使用する代替策を取る
		return getGitHubRepoInfoFallback(ctx)
	}

	// 現在の作業ディレクトリを取得
	cwd, err := os.Getwd()
	if err != nil {
		return nil, &GetGitHubRepoInfoError{
			Step:    "working_directory",
			Cause:   err,
			Message: "作業ディレクトリの取得に失敗しました",
		}
	}

	// .gitディレクトリを探す
	gitDir := findGitDirectory(cwd)
	if gitDir == "" {
		return nil, &GetGitHubRepoInfoError{
			Step:    "git_directory",
			Cause:   fmt.Errorf("no .git directory found"),
			Message: "Gitリポジトリが見つかりません。Gitリポジトリのルートディレクトリで実行してください",
		}
	}

	// リポジトリのルートディレクトリを取得
	repoRoot := filepath.Dir(gitDir)

	// git remote get-url origin を実行
	repo := git.NewRepository(log)
	remoteURL, err := repo.GetRemoteURL(ctx, repoRoot, "origin")
	if err != nil {
		return nil, &GetGitHubRepoInfoError{
			Step:    "remote_url",
			Cause:   err,
			Message: "リモートURL取得に失敗しました。'origin' リモートが設定されているか確認してください",
		}
	}

	// URLからowner/repo情報を抽出
	repoInfo, err := ParseGitHubURL(remoteURL)
	if err != nil {
		return nil, &GetGitHubRepoInfoError{
			Step:    "url_parsing",
			Cause:   err,
			Message: fmt.Sprintf("GitHub URL解析に失敗しました。URL: %s", remoteURL),
		}
	}

	return repoInfo, nil
}

// getGitHubRepoInfoFallback はロガー作成に失敗した場合のフォールバック実装
func getGitHubRepoInfoFallback(ctx context.Context) (*GitHubRepoInfo, error) {
	// 最小限の実装でリポジトリ情報を取得
	cwd, err := os.Getwd()
	if err != nil {
		return nil, &GetGitHubRepoInfoError{
			Step:    "working_directory",
			Cause:   err,
			Message: "作業ディレクトリの取得に失敗しました",
		}
	}

	gitDir := findGitDirectory(cwd)
	if gitDir == "" {
		return nil, &GetGitHubRepoInfoError{
			Step:    "git_directory",
			Cause:   fmt.Errorf("no .git directory found"),
			Message: "Gitリポジトリが見つかりません",
		}
	}

	// 簡易的なgit remoteコマンド実行
	// この部分は必要に応じて実装を追加
	return nil, &GetGitHubRepoInfoError{
		Step:    "fallback",
		Cause:   fmt.Errorf("logger creation failed and fallback not implemented"),
		Message: "システムエラーが発生しました",
	}
}

// findGitDirectory は指定されたパスから.gitディレクトリを探す
func findGitDirectory(startPath string) string {
	path := startPath
	for {
		gitPath := filepath.Join(path, ".git")
		if info, err := os.Stat(gitPath); err == nil {
			if info.IsDir() {
				return gitPath
			}
			// .gitがファイルの場合（worktreeの場合）
			// ファイルの内容を読んで実際の.gitディレクトリを見つける
			return gitPath
		}

		parent := filepath.Dir(path)
		if parent == path {
			break
		}
		path = parent
	}
	return ""
}

// GetOwnerAndRepoFromGitHubURL は GitHubURL から owner と repo を取得する簡易関数
// 後方互換性のために提供
func GetOwnerAndRepoFromGitHubURL(url string) (owner, repo string, err error) {
	repoInfo, err := ParseGitHubURL(url)
	if err != nil {
		return "", "", err
	}
	return repoInfo.Owner, repoInfo.Repo, nil
}
