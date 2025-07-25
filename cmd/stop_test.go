package cmd

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/daemon"
	"github.com/douhashi/osoba/internal/paths"
	"github.com/douhashi/osoba/internal/testutil/helpers"
)

func TestStopCmd(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		setupMock    func(t *testing.T) func()
		wantErr      bool
		wantContains []string
	}{
		{
			name: "正常系: プロセスの停止に成功",
			args: []string{},
			setupMock: func(t *testing.T) func() {
				mocker := helpers.NewFunctionMocker()

				// リポジトリ識別子の取得をモック
				mocker.MockFunc(&getRepoIdentifierFunc, func() (string, error) {
					return "test-owner-repo", nil
				})

				// リポジトリ名の取得をモック
				mocker.MockFunc(&getRepositoryNameFunc, func() (string, error) {
					return "test-repo", nil
				})

				// プロセス停止の成功をモック
				mocker.MockFunc(&stopProcessFunc, func(pidFile string) error {
					return nil
				})

				return mocker.Restore
			},
			wantErr: false,
			wantContains: []string{
				"停止しました",
			},
		},
		{
			name: "異常系: プロセスが実行されていないが処理は継続",
			args: []string{},
			setupMock: func(t *testing.T) func() {
				mocker := helpers.NewFunctionMocker()

				// リポジトリ識別子の取得をモック
				mocker.MockFunc(&getRepoIdentifierFunc, func() (string, error) {
					return "test-owner-repo", nil
				})

				// リポジトリ名の取得をモック
				mocker.MockFunc(&getRepositoryNameFunc, func() (string, error) {
					return "test-repo", nil
				})

				// プロセスが実行されていないことをモック
				mocker.MockFunc(&stopProcessFunc, func(pidFile string) error {
					return fmt.Errorf("プロセスが実行されていません")
				})

				// クリーンアップ処理をモック
				mocker.MockFunc(&performCleanupFunc, func(repoName string) error {
					return nil
				})

				// tmuxセッション削除をモック
				mocker.MockFunc(&killTmuxSessionFunc, func(sessionName string) error {
					return nil
				})

				return mocker.Restore
			},
			wantErr: false, // 新しい実装では部分的失敗でもエラーを返さない
			wantContains: []string{
				"プロセス停止に失敗しましたが、クリーンアップを継続します",
				"クリーンアップが完了しました",
				"tmuxセッションを削除しました",
			},
		},
		{
			name: "異常系: リポジトリ情報の取得に失敗",
			args: []string{},
			setupMock: func(t *testing.T) func() {
				mocker := helpers.NewFunctionMocker()

				// リポジトリ識別子の取得エラーをモック
				mocker.MockFunc(&getRepoIdentifierFunc, func() (string, error) {
					return "", fmt.Errorf("リポジトリ名の取得に失敗")
				})

				return mocker.Restore
			},
			wantErr: true,
			wantContains: []string{
				"リポジトリ名の取得に失敗",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックのセットアップ
			if tt.setupMock != nil {
				cleanup := tt.setupMock(t)
				defer cleanup()
			}

			// コマンドを実行
			output := &strings.Builder{}
			cmd := newStopCmd()
			cmd.SetArgs(tt.args)
			cmd.SetOut(output)
			cmd.SetErr(output)

			err := cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(output.String(), want) {
					t.Errorf("Execute() output = %v, want to contain %v", output.String(), want)
				}
			}
		})
	}
}

func TestStopCmd_WithCleanupAndTmuxKill(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		setupMock         func(t *testing.T) func()
		wantErr           bool
		wantContains      []string
		cleanupCalled     bool
		tmuxSessionKilled bool
	}{
		{
			name: "正常系: プロセス停止 + クリーンアップ + tmuxセッション削除に成功",
			args: []string{},
			setupMock: func(t *testing.T) func() {
				mocker := helpers.NewFunctionMocker()

				// リポジトリ識別子の取得をモック
				mocker.MockFunc(&getRepoIdentifierFunc, func() (string, error) {
					return "test-owner-repo", nil
				})

				// リポジトリ名の取得をモック
				mocker.MockFunc(&getRepositoryNameFunc, func() (string, error) {
					return "test-repo", nil
				})

				// プロセス停止の成功をモック
				mocker.MockFunc(&stopProcessFunc, func(pidFile string) error {
					return nil
				})

				// クリーンアップ処理をモック
				mocker.MockFunc(&performCleanupFunc, func(repoName string) error {
					return nil
				})

				// tmuxセッション削除をモック
				mocker.MockFunc(&killTmuxSessionFunc, func(sessionName string) error {
					return nil
				})

				return mocker.Restore
			},
			wantErr: false,
			wantContains: []string{
				"停止しました",
				"クリーンアップが完了しました",
				"tmuxセッションを削除しました",
			},
			cleanupCalled:     false,
			tmuxSessionKilled: false,
		},
		{
			name: "異常系: プロセス停止に失敗するが、クリーンアップとtmux削除は継続",
			args: []string{},
			setupMock: func(t *testing.T) func() {
				mocker := helpers.NewFunctionMocker()

				// リポジトリ識別子の取得をモック
				mocker.MockFunc(&getRepoIdentifierFunc, func() (string, error) {
					return "test-owner-repo", nil
				})

				// リポジトリ名の取得をモック
				mocker.MockFunc(&getRepositoryNameFunc, func() (string, error) {
					return "test-repo", nil
				})

				// プロセス停止の失敗をモック
				mocker.MockFunc(&stopProcessFunc, func(pidFile string) error {
					return fmt.Errorf("プロセス停止に失敗")
				})

				// クリーンアップ処理をモック
				mocker.MockFunc(&performCleanupFunc, func(repoName string) error {
					return nil
				})

				// tmuxセッション削除をモック
				mocker.MockFunc(&killTmuxSessionFunc, func(sessionName string) error {
					return nil
				})

				return mocker.Restore
			},
			wantErr: false, // 部分的失敗時でも処理を継続
			wantContains: []string{
				"プロセス停止に失敗",
				"クリーンアップが完了しました",
				"tmuxセッションを削除しました",
			},
			cleanupCalled:     false,
			tmuxSessionKilled: false,
		},
		{
			name: "異常系: クリーンアップに失敗するが、tmux削除は継続",
			args: []string{},
			setupMock: func(t *testing.T) func() {
				mocker := helpers.NewFunctionMocker()

				// リポジトリ識別子の取得をモック
				mocker.MockFunc(&getRepoIdentifierFunc, func() (string, error) {
					return "test-owner-repo", nil
				})

				// リポジトリ名の取得をモック
				mocker.MockFunc(&getRepositoryNameFunc, func() (string, error) {
					return "test-repo", nil
				})

				// プロセス停止の成功をモック
				mocker.MockFunc(&stopProcessFunc, func(pidFile string) error {
					return nil
				})

				// クリーンアップ処理の失敗をモック
				mocker.MockFunc(&performCleanupFunc, func(repoName string) error {
					return fmt.Errorf("クリーンアップに失敗")
				})

				// tmuxセッション削除をモック
				mocker.MockFunc(&killTmuxSessionFunc, func(sessionName string) error {
					return nil
				})

				return mocker.Restore
			},
			wantErr: false, // 部分的失敗時でも処理を継続
			wantContains: []string{
				"停止しました",
				"クリーンアップに失敗",
				"tmuxセッションを削除しました",
			},
			cleanupCalled:     false,
			tmuxSessionKilled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックのセットアップ
			if tt.setupMock != nil {
				cleanup := tt.setupMock(t)
				defer cleanup()
			}

			// コマンドを実行
			output := &strings.Builder{}
			cmd := newStopCmd()
			cmd.SetArgs(tt.args)
			cmd.SetOut(output)
			cmd.SetErr(output)

			err := cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(output.String(), want) {
					t.Errorf("Execute() output = %v, want to contain %v", output.String(), want)
				}
			}
		})
	}
}

func TestStopCmd_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		setup   func() (string, func())
		wantErr bool
	}{
		{
			name: "正常系: 実行中のプロセスを停止",
			setup: func() (string, func()) {
				// パスマネージャのモック
				pm := paths.NewPathManager(tmpDir)
				pm.EnsureDirectories()
				pidFile := pm.PIDFile("test-owner-repo")

				// テスト用のPIDファイルを作成（存在しないPIDを使用）
				info := &daemon.ProcessInfo{
					PID:       99999, // 存在しないPID
					StartTime: time.Now(),
					RepoPath:  "/path/to/repo",
				}
				daemon.WritePIDFile(pidFile, info)

				cleanup := func() {
					os.Remove(pidFile)
				}

				return pidFile, cleanup
			},
			wantErr: false,
		},
		{
			name: "異常系: PIDファイルが存在しない",
			setup: func() (string, func()) {
				pm := paths.NewPathManager(tmpDir)
				pidFile := pm.PIDFile("nonexistent-repo")

				cleanup := func() {}
				return pidFile, cleanup
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pidFile, cleanup := tt.setup()
			defer cleanup()

			dm := daemon.NewDaemonManager()
			err := dm.Stop(pidFile)

			if tt.name == "正常系: 実行中のプロセスを停止" {
				// 存在しないプロセスのため、実際にはプロセスが見つからない
				// PIDファイルはクリーンアップされるはず
				if err != nil {
					t.Errorf("Stop() error = %v", err)
				}

				// PIDファイルが削除されたことを確認
				if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
					t.Error("PID file was not cleaned up")
				}
			} else if tt.wantErr && err == nil {
				t.Error("Stop() expected error but got nil")
			}
		})
	}
}
