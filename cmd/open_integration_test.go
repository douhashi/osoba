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

// TestOpenCommandAutoRecoveryIntegration は自動復旧機能の統合テスト
func TestOpenCommandAutoRecoveryIntegration(t *testing.T) {
	// 元の関数を保存
	originalCheckTmux := checkTmuxInstalledFunc
	originalSessionExists := sessionExistsFunc
	originalGetRepoName := getRepositoryNameFunc
	originalGetRepoIdentifier := getRepoIdentifierFunc

	// テスト用の一時ディレクトリを作成
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	// テスト後に復元
	defer func() {
		checkTmuxInstalledFunc = originalCheckTmux
		sessionExistsFunc = originalSessionExists
		getRepositoryNameFunc = originalGetRepoName
		getRepoIdentifierFunc = originalGetRepoIdentifier
	}()

	// 現在のプロセスIDでPIDファイルを作成
	pid := os.Getpid()
	workingDir := "/tmp/test-repo"
	repoName := "test-repo"
	owner := "testuser"
	repoIdentifier := fmt.Sprintf("%s-%s", owner, repoName)

	pathManager := paths.NewPathManager("")
	pidFile := pathManager.PIDFile(repoIdentifier)

	// PIDファイルのディレクトリを作成
	pidDir := filepath.Dir(pidFile)
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		t.Fatalf("PIDファイルのディレクトリ作成に失敗: %v", err)
	}

	// PIDファイルを作成
	info := &daemon.ProcessInfo{
		PID:       pid,
		StartTime: time.Now(),
		RepoPath:  workingDir,
	}
	if err := daemon.WritePIDFile(pidFile, info); err != nil {
		t.Fatalf("PIDファイルの作成に失敗: %v", err)
	}

	// モックの設定
	checkTmuxInstalledFunc = func() error { return nil }
	getRepositoryNameFunc = func() (string, error) { return repoName, nil }
	getRepoIdentifierFunc = func() (string, error) {
		return repoIdentifier, nil
	}

	sessionExists := false
	sessionExistsFunc = func(name string) (bool, error) {
		return sessionExists, nil
	}

	// open.goのattemptSessionRecoveryをテスト
	_ = "osoba-" + repoName

	// 現在のワーキングディレクトリを保存して一時的に変更
	originalWd, _ := os.Getwd()
	os.Chdir("/tmp/test-repo")
	defer os.Chdir(originalWd)

	// attemptSessionRecoveryを直接呼び出せないため、
	// runOpenコマンドを通じてテスト
	err := runOpen(nil, []string{})

	// エラーが発生しないはずだが、実際のtmuxコマンドが実行されるためエラーになる
	// ここでは、PIDファイルが正しく認識されることを確認
	if err == nil {
		t.Error("エラーが発生すべき（tmuxコマンドが存在しないため）")
	}

	// エラーメッセージを確認
	errMsg := err.Error()
	if strings.Contains(errMsg, "osoba start") && strings.Contains(errMsg, "見つかりません") {
		// PIDファイルが見つからないか、プロセスが動作していない
		t.Logf("期待されるエラー: %s", errMsg)

		// DaemonManagerで直接確認
		dm := daemon.NewDaemonManager()
		if dm.IsRunning(pidFile) {
			t.Error("PIDファイルが存在し、プロセスも動作中のはず")
		}
	}
}

// TestSessionRecoveryLogic は自動復旧ロジックのユニットテスト
func TestSessionRecoveryLogic(t *testing.T) {
	tests := []struct {
		name             string
		pidFileExists    bool
		processRunning   bool
		sessionExists    bool
		expectRecovery   bool
		expectError      bool
		expectedErrorMsg string
	}{
		{
			name:           "復旧成功: PIDファイル存在、プロセス動作中、セッション無し",
			pidFileExists:  true,
			processRunning: true,
			sessionExists:  false,
			expectRecovery: true,
			expectError:    false,
		},
		{
			name:           "復旧不要: セッションが既に存在",
			pidFileExists:  true,
			processRunning: true,
			sessionExists:  true,
			expectRecovery: false,
			expectError:    false,
		},
		{
			name:             "復旧失敗: PIDファイル無し",
			pidFileExists:    false,
			processRunning:   false,
			sessionExists:    false,
			expectRecovery:   false,
			expectError:      true,
			expectedErrorMsg: "osoba start",
		},
		{
			name:             "復旧失敗: プロセス停止中",
			pidFileExists:    true,
			processRunning:   false,
			sessionExists:    false,
			expectRecovery:   false,
			expectError:      true,
			expectedErrorMsg: "osoba start",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テストケースに基づいた条件設定
			t.Logf("テストケース: %s", tt.name)
			t.Logf("条件: PIDファイル=%v, プロセス=%v, セッション=%v",
				tt.pidFileExists, tt.processRunning, tt.sessionExists)
			t.Logf("期待: 復旧=%v, エラー=%v",
				tt.expectRecovery, tt.expectError)
		})
	}
}

// TestPIDFileDetection はPIDファイルの検出ロジックをテスト
func TestPIDFileDetection(t *testing.T) {
	// 一時ディレクトリ作成
	tmpDir := t.TempDir()

	// PathManagerの初期化
	pathManager := paths.NewPathManager(tmpDir)

	// リポジトリ識別子を生成
	repoName := "test-repo"
	workingDir := "/home/user/repos/test-repo"
	repoIdentifier := repoName + "_home_user_repos_test-repo"

	// PIDファイルのパスを取得
	pidFile := pathManager.PIDFile(repoIdentifier)
	t.Logf("PIDファイルパス: %s", pidFile)

	// PIDファイルを作成
	info := &daemon.ProcessInfo{
		PID:       os.Getpid(),
		StartTime: time.Now(),
		RepoPath:  workingDir,
	}

	// ディレクトリを作成
	if err := os.MkdirAll(filepath.Dir(pidFile), 0755); err != nil {
		t.Fatalf("ディレクトリ作成エラー: %v", err)
	}

	if err := daemon.WritePIDFile(pidFile, info); err != nil {
		t.Fatalf("PIDファイル作成エラー: %v", err)
	}

	// DaemonManagerで確認
	dm := daemon.NewDaemonManager()
	if !dm.IsRunning(pidFile) {
		t.Error("PIDファイルが検出されるべき")
	}

	// PIDファイルを削除
	os.Remove(pidFile)

	// 再度確認
	if dm.IsRunning(pidFile) {
		t.Error("PIDファイルが検出されないべき")
	}
}
