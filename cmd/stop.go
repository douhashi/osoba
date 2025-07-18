package cmd

import (
	"fmt"
	"os/exec"

	"github.com/douhashi/osoba/internal/daemon"
	"github.com/douhashi/osoba/internal/paths"
	"github.com/douhashi/osoba/internal/tmux"
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
var (
	stopProcessFunc     = stopProcess
	performCleanupFunc  = performCleanup
	killTmuxSessionFunc = killTmuxSession
)

func runStop(cmd *cobra.Command, args []string) error {
	// リポジトリ識別子を取得
	repoIdentifier, err := getRepoIdentifierFunc()
	if err != nil {
		return err
	}

	// リポジトリ名を取得（クリーンアップとtmuxセッション削除に使用）
	repoName, err := getRepositoryNameFunc()
	if err != nil {
		return fmt.Errorf("リポジトリ名の取得に失敗: %w", err)
	}

	// パスマネージャを作成
	pm := paths.NewPathManager("")
	pidFile := pm.PIDFile(repoIdentifier)

	// エラーを収集（部分的失敗時でも処理を継続）
	var errors []error

	// 1. プロセスを停止
	if err := stopProcessFunc(pidFile); err != nil {
		errors = append(errors, fmt.Errorf("プロセス停止に失敗: %w", err))
		fmt.Fprintf(cmd.OutOrStderr(), "プロセス停止に失敗しましたが、クリーンアップを継続します: %v\n", err)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Issue監視を停止しました。\n")
	}

	// 2. クリーンアップ処理（clean --all --force 相当）
	sessionName := fmt.Sprintf("osoba-%s", repoName)
	if err := performCleanupFunc(sessionName); err != nil {
		errors = append(errors, fmt.Errorf("クリーンアップに失敗: %w", err))
		fmt.Fprintf(cmd.OutOrStderr(), "クリーンアップに失敗しましたが、tmux削除を継続します: %v\n", err)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "クリーンアップが完了しました。\n")
	}

	// 3. tmuxセッションを削除
	if err := killTmuxSessionFunc(sessionName); err != nil {
		errors = append(errors, fmt.Errorf("tmuxセッション削除に失敗: %w", err))
		fmt.Fprintf(cmd.OutOrStderr(), "tmuxセッション削除に失敗しました: %v\n", err)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "tmuxセッションを削除しました。\n")
	}

	// 最終メッセージ
	if len(errors) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "停止処理が完了しました。\n")
	} else {
		fmt.Fprintf(cmd.OutOrStderr(), "停止処理が完了しましたが、%d個のエラーが発生しました。\n", len(errors))
	}

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

// performCleanup は clean --all --force 相当の処理を実行します
func performCleanup(sessionName string) error {
	return performCleanupAllForce(sessionName)
}

// killTmuxSession は指定されたtmuxセッションを削除します
func killTmuxSession(sessionName string) error {
	// tmuxがインストールされているか確認
	if err := tmux.CheckTmuxInstalled(); err != nil {
		return fmt.Errorf("tmuxが利用できません: %w", err)
	}

	// セッションが存在するか確認
	exists, err := tmux.SessionExists(sessionName)
	if err != nil {
		return fmt.Errorf("セッション存在確認に失敗: %w", err)
	}

	if !exists {
		// セッションが存在しない場合は正常終了（削除済みと見なす）
		return nil
	}

	// セッションを削除（kill-sessionコマンドを直接実行）
	if err := killSessionWithCommand(sessionName); err != nil {
		return fmt.Errorf("セッション削除に失敗: %w", err)
	}

	return nil
}

// killSessionWithCommand はtmux kill-sessionコマンドを実行します
func killSessionWithCommand(sessionName string) error {
	cmd := stopExecCommand("tmux", "kill-session", "-t", sessionName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tmuxコマンド実行失敗: %w", err)
	}
	return nil
}

// テスト用にstopExecCommandを定義
var stopExecCommand = func(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
