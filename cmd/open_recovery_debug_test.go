package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/daemon"
	"github.com/douhashi/osoba/internal/paths"
)

// TestRepoIdentifierGeneration はリポジトリ識別子の生成ロジックをテスト
func TestRepoIdentifierGeneration(t *testing.T) {
	tests := []struct {
		name       string
		repoName   string
		workingDir string
		expectedID string
	}{
		{
			name:       "通常のパス",
			repoName:   "osoba",
			workingDir: "/home/user/repos/osoba",
			expectedID: "osoba__home_user_repos_osoba",
		},
		{
			name:       "深いパス",
			repoName:   "test-repo",
			workingDir: "/home/user/go/src/github.com/user/test-repo",
			expectedID: "test-repo__home_user_go_src_github.com_user_test-repo",
		},
		{
			name:       "ルートに近いパス",
			repoName:   "project",
			workingDir: "/tmp/project",
			expectedID: "project__tmp_project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// open.goの実装と同じロジック
			repoIdentifier := tt.repoName
			if tt.workingDir != "" {
				repoIdentifier = fmt.Sprintf("%s_%s", tt.repoName, strings.ReplaceAll(tt.workingDir, "/", "_"))
			}

			if repoIdentifier != tt.expectedID {
				t.Errorf("期待値と異なる識別子\n期待: %s\n実際: %s", tt.expectedID, repoIdentifier)
			}
		})
	}
}

// TestAttemptSessionRecoveryDebug は自動復旧機能の詳細なデバッグテスト
func TestAttemptSessionRecoveryDebug(t *testing.T) {
	// テスト用の一時ディレクトリ
	tmpDir := t.TempDir()

	// テスト用のリポジトリ情報
	repoName := "osoba"
	workingDir := "/home/douhashi/workspace/github.com/douhashi/osoba"

	// 1. 実際の識別子生成をシミュレート
	repoIdentifier := repoName
	if workingDir != "" {
		repoIdentifier = fmt.Sprintf("%s_%s", repoName, strings.ReplaceAll(workingDir, "/", "_"))
	}
	t.Logf("生成された識別子: %s", repoIdentifier)

	// 2. PathManagerを使ってPIDファイルパスを取得
	pathManager := paths.NewPathManager(tmpDir)
	pidFile := pathManager.PIDFile(repoIdentifier)
	t.Logf("PIDファイルパス: %s", pidFile)

	// 3. PIDファイルを作成
	pidDir := filepath.Dir(pidFile)
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		t.Fatalf("ディレクトリ作成失敗: %v", err)
	}

	info := &daemon.ProcessInfo{
		PID:       os.Getpid(),
		StartTime: time.Now(),
		RepoPath:  workingDir,
	}

	if err := daemon.WritePIDFile(pidFile, info); err != nil {
		t.Fatalf("PIDファイル作成失敗: %v", err)
	}

	// 4. DaemonManagerで確認
	dm := daemon.NewDaemonManager()
	isRunning := dm.IsRunning(pidFile)
	t.Logf("デーモン動作状態: %v", isRunning)

	if !isRunning {
		t.Error("デーモンが動作中として認識されるべき")

		// 詳細な診断
		if _, err := os.Stat(pidFile); os.IsNotExist(err) {
			t.Error("PIDファイルが存在しない")
		} else {
			t.Log("PIDファイルは存在する")

			// PIDファイルの内容を確認
			content, _ := os.ReadFile(pidFile)
			t.Logf("PIDファイルの内容:\n%s", string(content))

			// ProcessInfoを読み込んで確認
			if readInfo, err := daemon.ReadPIDFile(pidFile); err != nil {
				t.Errorf("PIDファイル読み込みエラー: %v", err)
			} else {
				t.Logf("読み込まれたPID: %d", readInfo.PID)
				t.Logf("現在のPID: %d", os.Getpid())
			}
		}
	}
}

// TestSessionRecoveryWithCorrectPath は正しいパスでの自動復旧をテスト
func TestSessionRecoveryWithCorrectPath(t *testing.T) {
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
	testWorkDir := filepath.Join(tmpDir, "repos", "test-repo")
	if err := os.MkdirAll(testWorkDir, 0755); err != nil {
		t.Fatalf("作業ディレクトリ作成失敗: %v", err)
	}

	// 現在のディレクトリを保存して変更
	originalWd, _ := os.Getwd()
	if err := os.Chdir(testWorkDir); err != nil {
		t.Fatalf("ディレクトリ変更失敗: %v", err)
	}
	defer os.Chdir(originalWd)

	// リポジトリ名とパス
	repoName := "test-repo"

	// 識別子を生成
	repoIdentifier := fmt.Sprintf("%s_%s", repoName, strings.ReplaceAll(testWorkDir, "/", "_"))
	t.Logf("識別子: %s", repoIdentifier)

	// PIDファイルを作成
	pathManager := paths.NewPathManager("")
	pidFile := pathManager.PIDFile(repoIdentifier)

	pidDir := filepath.Dir(pidFile)
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		t.Fatalf("PIDディレクトリ作成失敗: %v", err)
	}

	info := &daemon.ProcessInfo{
		PID:       os.Getpid(),
		StartTime: time.Now(),
		RepoPath:  testWorkDir,
	}

	if err := daemon.WritePIDFile(pidFile, info); err != nil {
		t.Fatalf("PIDファイル作成失敗: %v", err)
	}

	// モックの設定
	checkTmuxInstalledFunc = func() error { return nil }
	getRepositoryNameFunc = func() (string, error) { return repoName, nil }
	sessionExistsFunc = func(name string) (bool, error) {
		return false, nil // セッションが存在しない
	}

	// attemptSessionRecoveryをシミュレート
	sessionName := "osoba-" + repoName
	err := attemptSessionRecovery(sessionName, repoName)

	// tmuxコマンドが実際には存在しないため、エラーになる可能性があるが、
	// "osoba start"というエラーメッセージは出ないはず
	if err != nil {
		errMsg := err.Error()
		t.Logf("エラー: %s", errMsg)

		if strings.Contains(errMsg, "osoba start") {
			t.Error("PIDファイルが存在するのに 'osoba start' エラーが出た")
		}
	}
}
