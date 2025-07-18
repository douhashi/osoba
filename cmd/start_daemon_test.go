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
	"github.com/spf13/cobra"
)

func TestStartCmd_DaemonMode(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		envVars    map[string]string
		setupMock  func(t *testing.T) func()
		wantErr    bool
		wantOutput string
	}{
		{
			name: "正常系: デフォルトでバックグラウンド実行",
			args: []string{},
			setupMock: func(t *testing.T) func() {
				mocker := helpers.NewFunctionMocker()

				// ダemonモードではない場合の動作をモック
				mocker.MockFunc(&isDaemonModeFunc, func() bool {
					return false
				})

				// バックグラウンド起動の成功をモック
				mocker.MockFunc(&startInBackgroundFunc, func(cmd *cobra.Command, args []string) error {
					fmt.Fprintf(cmd.OutOrStdout(), "バックグラウンドで起動しました。PID: 12345\n")
					return nil
				})

				return mocker.Restore
			},
			wantErr:    false,
			wantOutput: "バックグラウンドで起動しました",
		},
		{
			name: "正常系: --foregroundフラグでフォアグラウンド実行",
			args: []string{"--foreground"},
			setupMock: func(t *testing.T) func() {
				mocker := helpers.NewFunctionMocker()

				// フォアグラウンドモードの場合は従来の処理をモック
				mocker.MockFunc(&runWatchWithFlagsFunc, func(cmd *cobra.Command, args []string, intervalFlag, configFlag string) error {
					fmt.Fprintf(cmd.OutOrStdout(), "Issue監視モードを開始します\n")
					return nil
				})

				return mocker.Restore
			},
			wantErr:    false,
			wantOutput: "Issue監視モードを開始します",
		},
		{
			name: "正常系: デーモンモードで起動された場合",
			args: []string{},
			envVars: map[string]string{
				"OSOBA_DAEMON_MODE": "1",
			},
			setupMock: func(t *testing.T) func() {
				mocker := helpers.NewFunctionMocker()

				// デーモンモードの場合の動作をモック
				mocker.MockFunc(&isDaemonModeFunc, func() bool {
					return true
				})

				// リポジトリ識別子の取得をモック
				mocker.MockFunc(&getRepoIdentifierFunc, func() (string, error) {
					return "test-owner-repo", nil
				})

				// デーモンモードでの実行をモック
				mocker.MockFunc(&runInDaemonModeFunc, func(cmd *cobra.Command, pidFile string, intervalFlag, configFlag string) error {
					fmt.Fprintf(cmd.OutOrStdout(), "デーモンモードで実行中\n")
					return nil
				})

				return mocker.Restore
			},
			wantErr:    false,
			wantOutput: "デーモンモードで実行中",
		},
		{
			name: "異常系: すでに実行中のプロセスがある",
			args: []string{},
			setupMock: func(t *testing.T) func() {
				mocker := helpers.NewFunctionMocker()

				// ダemonモードではない
				mocker.MockFunc(&isDaemonModeFunc, func() bool {
					return false
				})

				// リポジトリ識別子の取得をモック
				mocker.MockFunc(&getRepoIdentifierFunc, func() (string, error) {
					return "test-owner-repo", nil
				})

				// 既存プロセスのチェックをモック
				mocker.MockFunc(&checkExistingProcessFunc, func(pidFile string) (bool, error) {
					return true, nil // プロセスが存在する
				})

				// バックグラウンド起動をモック（呼ばれないはず）
				mocker.MockFunc(&startInBackgroundFunc, func(cmd *cobra.Command, args []string) error {
					return fmt.Errorf("すでに実行中です")
				})

				return mocker.Restore
			},
			wantErr:    true,
			wantOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数の設定
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// モックのセットアップ
			if tt.setupMock != nil {
				cleanup := tt.setupMock(t)
				defer cleanup()
			}

			// コマンドを実行
			output := &strings.Builder{}
			cmd := newStartCmd()
			cmd.SetArgs(tt.args)
			cmd.SetOut(output)
			cmd.SetErr(output)

			err := cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantOutput != "" && !strings.Contains(output.String(), tt.wantOutput) {
				t.Errorf("Execute() output = %v, want to contain %v", output.String(), tt.wantOutput)
			}
		})
	}
}

func TestStartCmd_PIDFileManagement(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		setupFiles func() string
		setupMock  func(t *testing.T) func()
		wantErr    bool
	}{
		{
			name: "正常系: PIDファイルが正しく作成される",
			setupFiles: func() string {
				// パスマネージャのモック
				pm := paths.NewPathManager(tmpDir)
				pm.EnsureDirectories()
				return pm.PIDFile("test-owner-repo")
			},
			setupMock: func(t *testing.T) func() {
				mocker := helpers.NewFunctionMocker()

				// PIDファイル作成の成功をモック
				mocker.MockFunc(&createPIDFileFunc, func(pidFile string) error {
					// 実際にPIDファイルを作成
					info := &daemon.ProcessInfo{
						PID:       os.Getpid(),
						StartTime: time.Now(),
						RepoPath:  "/path/to/repo",
					}
					return daemon.WritePIDFile(pidFile, info)
				})

				return mocker.Restore
			},
			wantErr: false,
		},
		{
			name: "異常系: PIDファイル作成に失敗",
			setupFiles: func() string {
				// 存在しないディレクトリを指定
				return "/nonexistent/dir/test.pid"
			},
			setupMock: func(t *testing.T) func() {
				mocker := helpers.NewFunctionMocker()

				// PIDファイル作成の失敗をモック
				mocker.MockFunc(&createPIDFileFunc, func(pidFile string) error {
					return fmt.Errorf("PIDファイルの作成に失敗")
				})

				return mocker.Restore
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pidFile := tt.setupFiles()

			if tt.setupMock != nil {
				cleanup := tt.setupMock(t)
				defer cleanup()
			}

			// PIDファイルが正しく作成されるかテスト
			err := createPIDFileFunc(pidFile)

			if (err != nil) != tt.wantErr {
				t.Errorf("createPIDFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// PIDファイルが存在することを確認
				if _, err := os.Stat(pidFile); os.IsNotExist(err) {
					t.Error("PID file was not created")
				}
			}
		})
	}
}
