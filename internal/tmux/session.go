package tmux

import (
	"errors"
	"fmt"
	"os/exec"
)

var (
	// ErrTmuxNotInstalled tmuxがインストールされていない場合のエラー
	ErrTmuxNotInstalled = errors.New("tmuxがインストールされていません。\n" +
		"インストール方法:\n" +
		"  Ubuntu/Debian: sudo apt-get install tmux\n" +
		"  macOS: brew install tmux\n" +
		"  その他: https://github.com/tmux/tmux/wiki/Installing")
)

// execCommand はテスト時にモック可能なコマンド実行関数
var execCommand = exec.Command

// CheckTmuxInstalled tmuxがインストールされているか確認
func CheckTmuxInstalled() error {
	cmd := execCommand("which", "tmux")
	if err := cmd.Run(); err != nil {
		return ErrTmuxNotInstalled
	}
	return nil
}

// SessionExists 指定された名前のtmuxセッションが存在するか確認
func SessionExists(sessionName string) (bool, error) {
	cmd := execCommand("tmux", "has-session", "-t", sessionName)
	err := cmd.Run()

	if err != nil {
		// tmuxのhas-sessionは、セッションが存在しない場合にエラーを返す
		if exitError, ok := err.(*exec.ExitError); ok {
			// 終了コード1はセッションが存在しないことを示す
			if exitError.ExitCode() == 1 {
				return false, nil
			}
		}
		// その他のエラー
		return false, fmt.Errorf("tmuxセッションの確認に失敗: %w", err)
	}

	return true, nil
}

// CreateSession 新しいtmuxセッションを作成
func CreateSession(sessionName string) error {
	// デタッチモードで新しいセッションを作成
	cmd := execCommand("tmux", "new-session", "-d", "-s", sessionName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tmuxセッションの作成に失敗: %w", err)
	}
	return nil
}
