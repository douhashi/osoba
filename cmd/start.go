package cmd

import (
	"errors"
	"fmt"

	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/tmux"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "tmuxセッションを作成",
		Long:  `現在のGitリポジトリ専用のtmuxセッションを作成します。`,
		RunE:  runStart,
	}
	return cmd
}

func runStart(cmd *cobra.Command, args []string) error {
	// 1. Gitリポジトリ名を取得
	repoName, err := git.GetRepositoryName()
	if err != nil {
		if errors.Is(err, git.ErrNotGitRepository) {
			return fmt.Errorf("%w", err)
		}
		if errors.Is(err, git.ErrNoRemoteFound) {
			return fmt.Errorf("%w", err)
		}
		return fmt.Errorf("リポジトリ名の取得に失敗: %w", err)
	}

	// 2. tmuxがインストールされているか確認
	if err := checkTmuxInstalled(); err != nil {
		return fmt.Errorf("%w", err)
	}

	// 3. セッション名を生成
	sessionName := fmt.Sprintf("osoba-%s", repoName)

	// 4. 既存セッションの確認
	exists, err := sessionExists(sessionName)
	if err != nil {
		return fmt.Errorf("セッションの確認に失敗: %w", err)
	}

	if exists {
		// 既存セッションがある場合
		fmt.Fprintf(cmd.OutOrStdout(), "tmuxセッション '%s' は既に存在します。\n", sessionName)
		fmt.Fprintf(cmd.OutOrStdout(), "接続するには以下のコマンドを実行してください:\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  tmux attach -t %s\n", sessionName)
		return nil
	}

	// 5. 新規セッションを作成
	if err := createSession(sessionName); err != nil {
		return fmt.Errorf("セッションの作成に失敗: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "tmuxセッション '%s' を作成しました。\n", sessionName)
	fmt.Fprintf(cmd.OutOrStdout(), "接続するには以下のコマンドを実行してください:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  tmux attach -t %s\n", sessionName)

	return nil
}

// テスト時にモック可能な関数変数
var (
	checkTmuxInstalled = tmux.CheckTmuxInstalled
	sessionExists      = tmux.SessionExists
	createSession      = tmux.CreateSession
)
