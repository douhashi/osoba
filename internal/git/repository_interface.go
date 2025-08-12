package git

import (
	"context"
	"strings"

	"github.com/douhashi/osoba/internal/logger"
)

// Repository はgitリポジトリ操作を管理するインターフェース
type Repository interface {
	// GetRootPath はリポジトリのルートパスを取得する
	GetRootPath(ctx context.Context) (string, error)

	// IsGitRepository は指定されたパスがgitリポジトリかを確認する
	IsGitRepository(ctx context.Context, path string) bool

	// GetCurrentCommit は現在のコミットハッシュを取得する
	GetCurrentCommit(ctx context.Context, path string) (string, error)

	// GetRemoteURL は指定されたリモートのURLを取得する
	GetRemoteURL(ctx context.Context, path string, remoteName string) (string, error)

	// GetStatus はリポジトリのステータスを取得する
	GetStatus(ctx context.Context, path string) (*RepositoryStatus, error)

	// GetLogger はロガーを取得する
	GetLogger() logger.Logger
}

// RepositoryStatus はリポジトリのステータス情報
type RepositoryStatus struct {
	IsClean        bool
	ModifiedFiles  []string
	UntrackedFiles []string
	StagedFiles    []string
}

// repositoryImpl はRepositoryインターフェースの実装
type repositoryImpl struct {
	logger  logger.Logger
	command *Command
}

// NewRepository は新しいRepositoryインスタンスを作成する
func NewRepository(logger logger.Logger) Repository {
	return &repositoryImpl{
		logger:  logger,
		command: NewCommand(logger),
	}
}

// GetRootPath はリポジトリのルートパスを取得する
func (r *repositoryImpl) GetRootPath(ctx context.Context) (string, error) {
	// git rev-parse --show-toplevelを実行
	output, err := r.command.Run(ctx, "git", []string{"rev-parse", "--show-toplevel"}, ".")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(output), nil
}

// IsGitRepository は指定されたパスがgitリポジトリかを確認する
func (r *repositoryImpl) IsGitRepository(ctx context.Context, path string) bool {
	// git rev-parse --git-dirを実行
	_, err := r.command.Run(ctx, "git", []string{"rev-parse", "--git-dir"}, path)
	return err == nil
}

// GetCurrentCommit は現在のコミットハッシュを取得する
func (r *repositoryImpl) GetCurrentCommit(ctx context.Context, path string) (string, error) {
	// git rev-parse HEADを実行
	output, err := r.command.Run(ctx, "git", []string{"rev-parse", "HEAD"}, path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(output), nil
}

// GetRemoteURL は指定されたリモートのURLを取得する
func (r *repositoryImpl) GetRemoteURL(ctx context.Context, path string, remoteName string) (string, error) {
	// git remote get-url <remote>を実行
	output, err := r.command.Run(ctx, "git", []string{"remote", "get-url", remoteName}, path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(output), nil
}

// GetStatus はリポジトリのステータスを取得する
func (r *repositoryImpl) GetStatus(ctx context.Context, path string) (*RepositoryStatus, error) {
	// git status --porcelainを実行
	output, err := r.command.Run(ctx, "git", []string{"status", "--porcelain"}, path)
	if err != nil {
		return nil, err
	}

	status := &RepositoryStatus{
		IsClean:        true,
		ModifiedFiles:  []string{},
		UntrackedFiles: []string{},
		StagedFiles:    []string{},
	}

	if output == "" {
		return status, nil
	}

	status.IsClean = false
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		statusCode := line[:2]
		filename := strings.TrimSpace(line[3:])

		switch {
		case statusCode == "??":
			status.UntrackedFiles = append(status.UntrackedFiles, filename)
		case statusCode[0] == 'M' || statusCode[1] == 'M':
			status.ModifiedFiles = append(status.ModifiedFiles, filename)
		case statusCode[0] != ' ' && statusCode[0] != '?':
			status.StagedFiles = append(status.StagedFiles, filename)
		}
	}

	return status, nil
}

// GetLogger はロガーを取得する
func (r *repositoryImpl) GetLogger() logger.Logger {
	return r.logger
}
