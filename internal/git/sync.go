package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/douhashi/osoba/internal/logger"
)

// RemoteInfo はリモートリポジトリの情報を表す構造体
type RemoteInfo struct {
	Name string
	URL  string
}

// StatusInfo はgitステータスの情報を表す構造体
type StatusInfo struct {
	IsClean        bool
	ModifiedFiles  []string
	StagedFiles    []string
	UntrackedFiles []string
	DeletedFiles   []string
}

// Sync はgit同期操作を管理する構造体
type Sync struct {
	logger  logger.Logger
	command *Command
}

// NewSync は新しいSyncインスタンスを作成する
func NewSync(logger logger.Logger) *Sync {
	return &Sync{
		logger:  logger,
		command: NewCommand(logger),
	}
}

// Fetch はリモートから変更を取得する
func (s *Sync) Fetch(ctx context.Context, repoPath, remote string, prune bool) error {
	logFields := []interface{}{
		"repoPath", repoPath,
		"remote", remote,
		"prune", prune,
	}

	s.logger.Info("Fetching from remote", logFields...)

	// fetchコマンドを構築
	args := []string{"fetch", remote}
	if prune {
		args = append(args, "--prune")
	}

	// fetchを実行
	output, err := s.command.Run(ctx, "git", args, repoPath)
	if err != nil {
		errorFields := append(logFields, "error", err.Error())
		s.logger.Error("Failed to fetch from remote", errorFields...)
		return fmt.Errorf("failed to fetch: %w", err)
	}

	// 成功ログ
	successFields := append(logFields, "output", output)
	s.logger.Info("Fetched from remote successfully", successFields...)

	return nil
}

// Pull はリモートから変更を取得してマージする
func (s *Sync) Pull(ctx context.Context, repoPath, remote, branch string, rebase bool) error {
	logFields := []interface{}{
		"repoPath", repoPath,
		"remote", remote,
		"branch", branch,
		"rebase", rebase,
	}

	s.logger.Info("Pulling from remote", logFields...)

	// pullコマンドを構築
	args := []string{"pull", remote, branch}
	if rebase {
		args = append(args, "--rebase")
	}

	// pullを実行
	output, err := s.command.Run(ctx, "git", args, repoPath)
	if err != nil {
		errorFields := append(logFields, "error", err.Error())
		s.logger.Error("Failed to pull from remote", errorFields...)

		// マージコンフリクトの可能性をチェック
		if strings.Contains(err.Error(), "conflict") || strings.Contains(err.Error(), "CONFLICT") {
			s.logger.Warn("Merge conflict detected", logFields...)
		}

		return fmt.Errorf("failed to pull: %w", err)
	}

	// 成功ログ
	successFields := append(logFields, "output", output)
	s.logger.Info("Pulled from remote successfully", successFields...)

	return nil
}

// Push はローカルの変更をリモートにプッシュする
func (s *Sync) Push(ctx context.Context, repoPath, remote, branch string, force, setUpstream bool) error {
	logFields := []interface{}{
		"repoPath", repoPath,
		"remote", remote,
		"branch", branch,
		"force", force,
		"setUpstream", setUpstream,
	}

	s.logger.Info("Pushing to remote", logFields...)

	// pushコマンドを構築
	args := []string{"push"}
	if setUpstream {
		args = append(args, "-u")
	}
	if force {
		args = append(args, "--force")
	}
	args = append(args, remote, branch)

	// pushを実行
	output, err := s.command.Run(ctx, "git", args, repoPath)
	if err != nil {
		errorFields := append(logFields, "error", err.Error())
		s.logger.Error("Failed to push to remote", errorFields...)

		// プッシュ拒否の可能性をチェック
		if strings.Contains(err.Error(), "rejected") {
			s.logger.Warn("Push rejected by remote", logFields...)
		}

		return fmt.Errorf("failed to push: %w", err)
	}

	// 成功ログ
	successFields := append(logFields, "output", output)
	s.logger.Info("Pushed to remote successfully", successFields...)

	return nil
}

// GetRemotes は設定されているリモートリポジトリの一覧を取得する
func (s *Sync) GetRemotes(ctx context.Context, repoPath string) ([]RemoteInfo, error) {
	logFields := []interface{}{
		"repoPath", repoPath,
	}

	s.logger.Info("Listing git remotes", logFields...)

	// remote -vを実行
	output, err := s.command.Run(ctx, "git", []string{"remote", "-v"}, repoPath)
	if err != nil {
		errorFields := append(logFields, "error", err.Error())
		s.logger.Error("Failed to list git remotes", errorFields...)
		return nil, fmt.Errorf("failed to list remotes: %w", err)
	}

	// 出力をパース
	remotes := parseRemoteList(output)

	// 成功ログ
	successFields := append(logFields, "count", len(remotes))
	s.logger.Info("Git remotes listed successfully", successFields...)

	// 各リモートの詳細をデバッグログに出力
	for i, remote := range remotes {
		s.logger.Debug("Remote info",
			"index", i,
			"name", remote.Name,
			"url", remote.URL,
		)
	}

	return remotes, nil
}

// parseRemoteList はgit remote -vの出力をパースする
func parseRemoteList(output string) []RemoteInfo {
	remoteMap := make(map[string]string)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// フォーマット: origin	https://github.com/user/repo.git (fetch)
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			name := parts[0]
			url := parts[1]
			// 同じリモートでfetchとpushの2行があるので、重複を避ける
			remoteMap[name] = url
		}
	}

	// マップからスライスに変換
	var remotes []RemoteInfo
	for name, url := range remoteMap {
		remotes = append(remotes, RemoteInfo{
			Name: name,
			URL:  url,
		})
	}

	return remotes
}

// GetStatus は作業ディレクトリのステータスを取得する
func (s *Sync) GetStatus(ctx context.Context, repoPath string) (*StatusInfo, error) {
	logFields := []interface{}{
		"repoPath", repoPath,
	}

	s.logger.Info("Getting git status", logFields...)

	// git status --porcelainを実行
	output, err := s.command.Run(ctx, "git", []string{"status", "--porcelain"}, repoPath)
	if err != nil {
		errorFields := append(logFields, "error", err.Error())
		s.logger.Error("Failed to get git status", errorFields...)
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	// 出力をパース
	status := parseStatusOutput(output)

	// ステータスログ
	statusFields := append(logFields,
		"isClean", status.IsClean,
		"modified", len(status.ModifiedFiles),
		"staged", len(status.StagedFiles),
		"untracked", len(status.UntrackedFiles),
		"deleted", len(status.DeletedFiles),
	)
	s.logger.Info("Git status retrieved", statusFields...)

	return status, nil
}

// parseStatusOutput はgit status --porcelainの出力をパースする
func parseStatusOutput(output string) *StatusInfo {
	status := &StatusInfo{
		IsClean:        true,
		ModifiedFiles:  []string{},
		StagedFiles:    []string{},
		UntrackedFiles: []string{},
		DeletedFiles:   []string{},
	}

	if output == "" {
		return status
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
		case statusCode[0] == 'M' || statusCode[1] == 'M':
			status.ModifiedFiles = append(status.ModifiedFiles, filename)
		case statusCode[0] == 'A':
			status.StagedFiles = append(status.StagedFiles, filename)
		case statusCode == "??":
			status.UntrackedFiles = append(status.UntrackedFiles, filename)
		case statusCode[0] == 'D' || statusCode[1] == 'D':
			status.DeletedFiles = append(status.DeletedFiles, filename)
		}

		// ステージングエリアにある変更
		if statusCode[0] != ' ' && statusCode[0] != '?' {
			if !contains(status.StagedFiles, filename) {
				status.StagedFiles = append(status.StagedFiles, filename)
			}
		}
	}

	return status
}

// contains はスライスに特定の文字列が含まれているかを確認する
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// HasRemote は指定されたリモートが存在するかを確認する
func (s *Sync) HasRemote(ctx context.Context, repoPath, remoteName string) bool {
	remotes, err := s.GetRemotes(ctx, repoPath)
	if err != nil {
		return false
	}

	for _, remote := range remotes {
		if remote.Name == remoteName {
			return true
		}
	}

	return false
}

// AddRemote は新しいリモートを追加する
func (s *Sync) AddRemote(ctx context.Context, repoPath, name, url string) error {
	logFields := []interface{}{
		"repoPath", repoPath,
		"name", name,
		"url", url,
	}

	s.logger.Info("Adding git remote", logFields...)

	// git remote add を実行
	args := []string{"remote", "add", name, url}
	output, err := s.command.Run(ctx, "git", args, repoPath)
	if err != nil {
		errorFields := append(logFields, "error", err.Error())
		s.logger.Error("Failed to add git remote", errorFields...)
		return fmt.Errorf("failed to add remote: %w", err)
	}

	// 成功ログ
	successFields := append(logFields, "output", output)
	s.logger.Info("Git remote added successfully", successFields...)

	return nil
}

// RemoveRemote は指定されたリモートを削除する
func (s *Sync) RemoveRemote(ctx context.Context, repoPath, name string) error {
	logFields := []interface{}{
		"repoPath", repoPath,
		"name", name,
	}

	s.logger.Info("Removing git remote", logFields...)

	// git remote remove を実行
	args := []string{"remote", "remove", name}
	output, err := s.command.Run(ctx, "git", args, repoPath)
	if err != nil {
		errorFields := append(logFields, "error", err.Error())
		s.logger.Error("Failed to remove git remote", errorFields...)
		return fmt.Errorf("failed to remove remote: %w", err)
	}

	// 成功ログ
	successFields := append(logFields, "output", output)
	s.logger.Info("Git remote removed successfully", successFields...)

	return nil
}

// FetchBranch は特定のブランチをリモートから直接フェッチする
func (s *Sync) FetchBranch(ctx context.Context, repoPath, remote, branch string) error {
	logFields := []interface{}{
		"repoPath", repoPath,
		"remote", remote,
		"branch", branch,
	}

	s.logger.Info("Fetching specific branch from remote", logFields...)

	// git fetch <remote> <branch>:<branch> を実行
	args := []string{"fetch", remote, fmt.Sprintf("%s:%s", branch, branch)}
	output, err := s.command.Run(ctx, "git", args, repoPath)
	if err != nil {
		errorFields := append(logFields, "error", err.Error())
		s.logger.Error("Failed to fetch branch from remote", errorFields...)
		return fmt.Errorf("failed to fetch branch: %w", err)
	}

	// 成功ログ
	successFields := append(logFields, "output", output)
	s.logger.Info("Branch fetched successfully", successFields...)

	return nil
}

// ResetHard はgit reset --hardを実行してローカル変更を破棄する
func (s *Sync) ResetHard(ctx context.Context, repoPath, ref string) error {
	logFields := []interface{}{
		"repoPath", repoPath,
		"ref", ref,
	}

	s.logger.Warn("Resetting working directory (discarding local changes)", logFields...)

	// git reset --hard <ref> を実行
	args := []string{"reset", "--hard", ref}
	output, err := s.command.Run(ctx, "git", args, repoPath)
	if err != nil {
		errorFields := append(logFields, "error", err.Error())
		s.logger.Error("Failed to reset working directory", errorFields...)
		return fmt.Errorf("failed to reset: %w", err)
	}

	// 成功ログ
	successFields := append(logFields, "output", output)
	s.logger.Info("Working directory reset successfully", successFields...)

	return nil
}
