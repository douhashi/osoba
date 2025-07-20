package helpers

import (
	"os/exec"
	"testing"
)

// InitGitRepo は指定されたディレクトリにGitリポジトリを初期化する
func InitGitRepo(t *testing.T, dir string) error {
	t.Helper()

	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return err
	}

	// 初期設定
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// SetGitRemote は指定されたリポジトリにリモートURLを設定する
func SetGitRemote(t *testing.T, dir, remoteName, remoteURL string) error {
	t.Helper()

	cmd := exec.Command("git", "remote", "add", remoteName, remoteURL)
	cmd.Dir = dir
	return cmd.Run()
}
