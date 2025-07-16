package tmux

import (
	"errors"
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
// 既存の関数はglobal_manager.goのラッパー関数に置き換えられたため、
// テストコードとの互換性を保つために残しています
var execCommand = exec.Command

// SessionInfo はtmuxセッションの情報を保持する構造体
type SessionInfo struct {
	Name     string
	Windows  int
	Created  string
	Attached bool
}
