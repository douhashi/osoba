package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/daemon"
	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/paths"
	"github.com/douhashi/osoba/internal/tmux"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newOpenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open",
		Short: "tmuxセッションに接続",
		Long:  `現在のGitリポジトリに対応するtmuxセッションに接続します。`,
		RunE:  runOpen,
	}
	return cmd
}

func runOpen(cmd *cobra.Command, args []string) error {
	// 1. tmuxがインストールされているか確認
	if err := checkTmuxInstalledFunc(); err != nil {
		return err
	}

	// 2. 設定を読み込み
	cfg := config.NewConfig()

	// rootコマンドで読み込まれた設定ファイルのパスを取得
	configPath := viper.ConfigFileUsed()
	if configPath == "" {
		// -cフラグが指定されている場合はそれを使用
		configPath = viper.GetString("config")
	}

	// 設定ファイルのパスが取得できた場合、またはデフォルトパスから読み込み
	if configPath != "" {
		cfg.LoadOrDefault(configPath)
	} else {
		// configPathが空の場合もデフォルト設定ファイルをチェック
		cfg.LoadOrDefault("")
	}

	// 3. Gitリポジトリ名を取得
	repoName, err := getRepositoryNameFunc()
	if err != nil {
		if errors.Is(err, git.ErrNotGitRepository) {
			return fmt.Errorf("現在のディレクトリはGitリポジトリではありません")
		}
		if errors.Is(err, git.ErrNoRemoteFound) {
			return fmt.Errorf("リモートリポジトリが設定されていません")
		}
		return fmt.Errorf("リポジトリ名の取得に失敗しました: %w", err)
	}

	// 4. セッション名を生成（設定から接頭辞を使用）
	sessionName := fmt.Sprintf("%s%s", cfg.Tmux.SessionPrefix, repoName)

	// 5. セッションが存在するか確認
	exists, err := sessionExistsFunc(sessionName)
	if err != nil {
		return fmt.Errorf("セッションの確認に失敗しました: %w", err)
	}

	if !exists {
		// セッション自動復旧を試行
		if err := attemptSessionRecovery(sessionName, repoName); err != nil {
			return err
		}
	}

	// 6. tmux内から実行されているか確認
	if isInsideTmux() {
		// tmux内からの場合は switch-client を使用
		return switchToSession(sessionName)
	}

	// 7. tmux外からの場合は attach を使用
	return attachToSession(sessionName)
}

// isInsideTmux はtmux内から実行されているかを確認
func isInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

// switchToSession はtmux内から別のセッションに切り替える
func switchToSession(sessionName string) error {
	cmd := exec.Command("tmux", "switch-client", "-t", sessionName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("セッションへの切り替えに失敗しました: %w", err)
	}
	return nil
}

// attachToSession はtmux外からセッションに接続する
func attachToSession(sessionName string) error {
	cmd := exec.Command("tmux", "attach-session", "-t", sessionName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// エラーメッセージから "exited" を除外（正常終了の場合）
		if exitErr, ok := err.(*exec.ExitError); ok {
			if !strings.Contains(exitErr.Error(), "exited") {
				return fmt.Errorf("セッションへの接続に失敗しました: %w", err)
			}
			// tmuxからのデタッチは正常終了として扱う
			return nil
		}
		return fmt.Errorf("セッションへの接続に失敗しました: %w", err)
	}
	return nil
}

// attemptSessionRecovery はセッション自動復旧を試行します
func attemptSessionRecovery(sessionName, repoName string) error {
	// 1. PathManagerを初期化
	pathManager := paths.NewPathManager("")

	// 2. start.goと同じ方式でリポジトリ識別子を生成
	repoIdentifier, err := getRepoIdentifierFunc()
	if err != nil {
		return fmt.Errorf("リポジトリ識別子の取得に失敗しました: %w", err)
	}

	// 3. PIDファイルのパスを取得
	pidFile := pathManager.PIDFile(repoIdentifier)

	// 4. DaemonManagerでosoba動作確認
	daemonManager := daemon.NewDaemonManager()
	isRunning := daemonManager.IsRunning(pidFile)

	if !isRunning {
		return fmt.Errorf("セッション '%s' が見つかりません。先に 'osoba start'を実行してください", sessionName)
	}

	// 5. tmuxセッションを再作成
	tmuxManager := tmux.NewDefaultManager()
	if err := tmuxManager.EnsureSession(sessionName); err != nil {
		return fmt.Errorf("セッションの復旧に失敗しました: %w", err)
	}

	return nil
}

// テスト時にモック可能な関数変数は clean.go で定義されています
