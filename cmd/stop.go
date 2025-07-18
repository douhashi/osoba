package cmd

import (
	"fmt"

	"github.com/douhashi/osoba/internal/daemon"
	"github.com/douhashi/osoba/internal/paths"
	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "バックグラウンドで実行中のIssue監視を停止",
		Long: `バックグラウンドで実行中のIssue監視プロセスを停止します。
現在のリポジトリに対応するプロセスのみを停止します。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStop(cmd, args)
		},
	}

	return cmd
}

// テスト用にモック可能な関数変数
var stopProcessFunc = stopProcess

func runStop(cmd *cobra.Command, args []string) error {
	// リポジトリ識別子を取得
	repoIdentifier, err := getRepoIdentifierFunc()
	if err != nil {
		return err
	}

	// パスマネージャを作成
	pm := paths.NewPathManager("")
	pidFile := pm.PIDFile(repoIdentifier)

	// プロセスを停止
	if err := stopProcessFunc(pidFile); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Issue監視を停止しました。\n")
	return nil
}

// stopProcess はプロセスを停止します
func stopProcess(pidFile string) error {
	dm := daemon.NewDaemonManager()

	// プロセスの状態を確認
	status, err := dm.Status(pidFile)
	if err != nil {
		return fmt.Errorf("プロセスの状態確認に失敗: %w", err)
	}

	if !status.Running {
		return fmt.Errorf("プロセスが実行されていません")
	}

	// プロセスを停止
	if err := dm.Stop(pidFile); err != nil {
		return fmt.Errorf("プロセスの停止に失敗: %w", err)
	}

	return nil
}
