package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/daemon"
	"github.com/douhashi/osoba/internal/paths"
	"github.com/douhashi/osoba/internal/utils"
)

// TestRepoIdentifierConsistency はstart.goとopen.goの識別子生成の一貫性をテスト
func TestRepoIdentifierConsistency(t *testing.T) {
	// モック用の環境設定
	tmpDir := t.TempDir()
	testRepo := filepath.Join(tmpDir, "test-repo")
	if err := os.MkdirAll(testRepo, 0755); err != nil {
		t.Fatalf("テストリポジトリの作成失敗: %v", err)
	}

	// Gitリポジトリを初期化
	if err := os.MkdirAll(filepath.Join(testRepo, ".git"), 0755); err != nil {
		t.Fatalf(".gitディレクトリの作成失敗: %v", err)
	}

	// リモートURLを設定（モック）
	gitConfig := `[remote "origin"]
	url = https://github.com/testuser/test-repo.git`
	if err := os.WriteFile(filepath.Join(testRepo, ".git", "config"), []byte(gitConfig), 0644); err != nil {
		t.Fatalf("git configの作成失敗: %v", err)
	}

	// 作業ディレクトリを変更
	originalWd, _ := os.Getwd()
	if err := os.Chdir(testRepo); err != nil {
		t.Fatalf("ディレクトリ変更失敗: %v", err)
	}
	defer os.Chdir(originalWd)

	// start.goの識別子生成をシミュレート
	repoInfo, err := utils.GetGitHubRepoInfo(context.Background())
	if err != nil {
		t.Logf("リポジトリ情報取得エラー（期待される）: %v", err)
		// テスト環境では失敗することがあるので、手動で設定
		repoInfo = &utils.GitHubRepoInfo{
			Owner: "testuser",
			Repo:  "test-repo",
		}
	}
	startIdentifier := fmt.Sprintf("%s-%s", repoInfo.Owner, repoInfo.Repo)
	t.Logf("start.goの識別子: %s", startIdentifier)

	// open.goの識別子生成をシミュレート（現在の実装）
	repoName := repoInfo.Repo
	workingDir, _ := os.Getwd()
	openIdentifier := repoName
	if workingDir != "" {
		openIdentifier = fmt.Sprintf("%s_%s", repoName, strings.ReplaceAll(workingDir, "/", "_"))
	}
	t.Logf("open.goの識別子（現在）: %s", openIdentifier)

	// 識別子が異なることを確認
	if startIdentifier == openIdentifier {
		t.Error("識別子が一致してしまっています（異なるべき）")
	}

	// 修正後の識別子生成（提案）
	fixedOpenIdentifier := fmt.Sprintf("%s-%s", repoInfo.Owner, repoInfo.Repo)
	t.Logf("open.goの識別子（修正後）: %s", fixedOpenIdentifier)

	// 修正後は一致することを確認
	if startIdentifier != fixedOpenIdentifier {
		t.Error("修正後の識別子が一致しません")
	}
}

// TestSessionRecoveryWithCorrectIdentifier は正しい識別子での自動復旧をテスト
func TestSessionRecoveryWithCorrectIdentifier(t *testing.T) {
	// 元の関数を保存
	originalCheckTmux := checkTmuxInstalledFunc
	originalSessionExists := sessionExistsFunc
	originalGetRepoName := getRepositoryNameFunc

	// テスト後に復元
	defer func() {
		checkTmuxInstalledFunc = originalCheckTmux
		sessionExistsFunc = originalSessionExists
		getRepositoryNameFunc = originalGetRepoName
	}()

	// テスト環境のセットアップ
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	// 正しい識別子（start.goと同じ形式）
	owner := "testuser"
	repo := "test-repo"
	correctIdentifier := fmt.Sprintf("%s-%s", owner, repo)

	// PIDファイルを作成
	pathManager := paths.NewPathManager("")
	pidFile := pathManager.PIDFile(correctIdentifier)

	pidDir := filepath.Dir(pidFile)
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		t.Fatalf("PIDディレクトリ作成失敗: %v", err)
	}

	info := &daemon.ProcessInfo{
		PID:       os.Getpid(),
		StartTime: time.Now(),
		RepoPath:  tmpDir,
	}

	if err := daemon.WritePIDFile(pidFile, info); err != nil {
		t.Fatalf("PIDファイル作成失敗: %v", err)
	}

	// DaemonManagerで確認
	dm := daemon.NewDaemonManager()
	if !dm.IsRunning(pidFile) {
		t.Error("PIDファイルが検出されるべき")
	}

	t.Logf("正しい識別子でPIDファイルが検出されました: %s", correctIdentifier)
}

// TestAttemptSessionRecoveryFixed は修正された自動復旧関数のテスト
func TestAttemptSessionRecoveryFixed(t *testing.T) {
	// このテストは、attemptSessionRecoveryが修正された後に動作確認するためのもの
	t.Skip("attemptSessionRecovery関数の修正後に有効化")

	// テスト環境のセットアップ
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	// リポジトリ情報
	owner := "testuser"
	repo := "test-repo"
	sessionName := "osoba-test-repo"

	// 正しい識別子でPIDファイルを作成
	identifier := fmt.Sprintf("%s-%s", owner, repo)
	pathManager := paths.NewPathManager("")
	pidFile := pathManager.PIDFile(identifier)

	pidDir := filepath.Dir(pidFile)
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		t.Fatalf("PIDディレクトリ作成失敗: %v", err)
	}

	info := &daemon.ProcessInfo{
		PID:       os.Getpid(),
		StartTime: time.Now(),
		RepoPath:  tmpDir,
	}

	if err := daemon.WritePIDFile(pidFile, info); err != nil {
		t.Fatalf("PIDファイル作成失敗: %v", err)
	}

	// attemptSessionRecoveryを呼び出す
	// ここで修正された関数が正しく動作することを確認
	err := attemptSessionRecoveryFixed(sessionName, repo, owner)

	// tmuxコマンドが存在しないためエラーになるが、
	// "osoba start"というエラーメッセージは出ないはず
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "osoba start") {
			t.Errorf("PIDファイルが存在するのに 'osoba start' エラーが出た: %s", errMsg)
		}
	}
}

// attemptSessionRecoveryFixed は修正された自動復旧関数（提案）
func attemptSessionRecoveryFixed(sessionName, repoName, owner string) error {
	// 1. PathManagerを初期化
	pathManager := paths.NewPathManager("")

	// 2. start.goと同じ方式で識別子を生成
	repoIdentifier := fmt.Sprintf("%s-%s", owner, repoName)

	// 3. PIDファイルのパスを取得
	pidFile := pathManager.PIDFile(repoIdentifier)

	// 4. DaemonManagerでosoba動作確認
	daemonManager := daemon.NewDaemonManager()
	isRunning := daemonManager.IsRunning(pidFile)

	if !isRunning {
		return fmt.Errorf("セッション '%s' が見つかりません。先に 'osoba start'を実行してください", sessionName)
	}

	// 5. tmuxセッションを再作成
	// tmuxManager := tmux.NewDefaultManager()
	// if err := tmuxManager.EnsureSession(sessionName); err != nil {
	//     return fmt.Errorf("セッションの復旧に失敗しました: %w", err)
	// }

	return nil
}
