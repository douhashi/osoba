package tmux

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestCheckTmuxInstalled(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func()
		wantErr     bool
		wantErrType error
	}{
		{
			name: "正常系: tmuxがインストールされている",
			setupMock: func() {
				execCommand = func(name string, arg ...string) *exec.Cmd {
					if name == "which" && len(arg) > 0 && arg[0] == "tmux" {
						return exec.Command("echo", "/usr/bin/tmux")
					}
					return exec.Command(name, arg...)
				}
			},
			wantErr: false,
		},
		{
			name: "異常系: tmuxがインストールされていない",
			setupMock: func() {
				execCommand = func(name string, arg ...string) *exec.Cmd {
					if name == "which" && len(arg) > 0 && arg[0] == "tmux" {
						return exec.Command("false")
					}
					return exec.Command(name, arg...)
				}
			},
			wantErr:     true,
			wantErrType: ErrTmuxNotInstalled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックのセットアップ
			tt.setupMock()
			defer func() {
				execCommand = exec.Command
			}()

			err := CheckTmuxInstalled()
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckTmuxInstalled() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErrType != nil && !errors.Is(err, tt.wantErrType) {
				t.Errorf("CheckTmuxInstalled() error = %v, wantErrType %v", err, tt.wantErrType)
			}
		})
	}
}

func TestSessionExists(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		setupMock   func()
		want        bool
		wantErr     bool
	}{
		{
			name:        "正常系: セッションが存在する",
			sessionName: "osoba-test",
			setupMock: func() {
				execCommand = func(name string, arg ...string) *exec.Cmd {
					if name == "tmux" && len(arg) >= 2 && arg[0] == "has-session" && arg[1] == "-t" {
						return exec.Command("true")
					}
					return exec.Command(name, arg...)
				}
			},
			want:    true,
			wantErr: false,
		},
		{
			name:        "正常系: セッションが存在しない",
			sessionName: "osoba-test",
			setupMock: func() {
				execCommand = func(name string, arg ...string) *exec.Cmd {
					if name == "tmux" && len(arg) >= 2 && arg[0] == "has-session" && arg[1] == "-t" {
						return exec.Command("false")
					}
					return exec.Command(name, arg...)
				}
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックのセットアップ
			tt.setupMock()
			defer func() {
				execCommand = exec.Command
			}()

			got, err := SessionExists(tt.sessionName)
			if (err != nil) != tt.wantErr {
				t.Errorf("SessionExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("SessionExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateSession(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		setupMock   func()
		wantErr     bool
	}{
		{
			name:        "正常系: セッション作成成功",
			sessionName: "osoba-test",
			setupMock: func() {
				execCommand = func(name string, arg ...string) *exec.Cmd {
					if name == "tmux" && len(arg) >= 3 && arg[0] == "new-session" {
						return exec.Command("true")
					}
					return exec.Command(name, arg...)
				}
			},
			wantErr: false,
		},
		{
			name:        "異常系: セッション作成失敗",
			sessionName: "osoba-test",
			setupMock: func() {
				execCommand = func(name string, arg ...string) *exec.Cmd {
					if name == "tmux" && len(arg) >= 3 && arg[0] == "new-session" {
						return exec.Command("false")
					}
					return exec.Command(name, arg...)
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックのセットアップ
			tt.setupMock()
			defer func() {
				execCommand = exec.Command
			}()

			err := CreateSession(tt.sessionName)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSession() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckTmuxInstalled_WithLogging(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func()
		wantLogMessage string
		wantLogLevel   string
	}{
		{
			name: "tmuxインストール確認時にログ出力される",
			setupMock: func() {
				execCommand = func(name string, arg ...string) *exec.Cmd {
					if name == "which" && len(arg) > 0 && arg[0] == "tmux" {
						return exec.Command("echo", "/usr/bin/tmux")
					}
					return exec.Command(name, arg...)
				}
			},
			wantLogMessage: "tmuxインストール確認",
			wantLogLevel:   "debug",
		},
		{
			name: "tmux未インストール時にエラーログ出力される",
			setupMock: func() {
				execCommand = func(name string, arg ...string) *exec.Cmd {
					if name == "which" && len(arg) > 0 && arg[0] == "tmux" {
						return exec.Command("false")
					}
					return exec.Command(name, arg...)
				}
			},
			wantLogMessage: "tmuxがインストールされていません",
			wantLogLevel:   "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックロガーのセットアップ
			mockLog := &mockLogger{}
			SetLogger(mockLog)
			defer SetLogger(nil)

			// コマンドモックのセットアップ
			tt.setupMock()
			defer func() {
				execCommand = exec.Command
			}()

			// 実行
			CheckTmuxInstalled()

			// ログ出力の検証
			switch tt.wantLogLevel {
			case "debug":
				if !containsMessage(mockLog.debugMessages, tt.wantLogMessage) {
					t.Errorf("期待するデバッグログが出力されませんでした: %s", tt.wantLogMessage)
				}
			case "error":
				if !containsMessage(mockLog.errorMessages, tt.wantLogMessage) {
					t.Errorf("期待するエラーログが出力されませんでした: %s", tt.wantLogMessage)
				}
			}
		})
	}
}

func TestSessionExists_WithLogging(t *testing.T) {
	tests := []struct {
		name           string
		sessionName    string
		exists         bool
		wantLogMessage string
	}{
		{
			name:           "セッション存在確認時にログ出力される",
			sessionName:    "test-session",
			exists:         true,
			wantLogMessage: "tmuxセッション確認",
		},
		{
			name:           "セッションが存在しない場合もログ出力される",
			sessionName:    "test-session",
			exists:         false,
			wantLogMessage: "tmuxセッション確認",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックロガーのセットアップ
			mockLog := &mockLogger{}
			SetLogger(mockLog)
			defer SetLogger(nil)

			// コマンドモックのセットアップ
			execCommand = func(name string, arg ...string) *exec.Cmd {
				if name == "tmux" && len(arg) >= 2 && arg[0] == "has-session" {
					if tt.exists {
						return exec.Command("true")
					}
					return exec.Command("false")
				}
				return exec.Command(name, arg...)
			}
			defer func() {
				execCommand = exec.Command
			}()

			// 実行
			SessionExists(tt.sessionName)

			// ログ出力の検証
			if !containsMessage(mockLog.debugMessages, tt.wantLogMessage) {
				t.Errorf("期待するログが出力されませんでした: %s", tt.wantLogMessage)
			}
		})
	}
}

func TestCreateSession_WithLogging(t *testing.T) {
	tests := []struct {
		name           string
		sessionName    string
		success        bool
		wantLogMessage string
		wantLogLevel   string
	}{
		{
			name:           "セッション作成開始時にログ出力される",
			sessionName:    "test-session",
			success:        true,
			wantLogMessage: "tmuxセッション作成開始",
			wantLogLevel:   "info",
		},
		{
			name:           "セッション作成失敗時にエラーログ出力される",
			sessionName:    "test-session",
			success:        false,
			wantLogMessage: "tmuxセッション作成失敗",
			wantLogLevel:   "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// モックロガーのセットアップ
			mockLog := &mockLogger{}
			SetLogger(mockLog)
			defer SetLogger(nil)

			// コマンドモックのセットアップ
			execCommand = func(name string, arg ...string) *exec.Cmd {
				if name == "tmux" && len(arg) >= 3 && arg[0] == "new-session" {
					if tt.success {
						return exec.Command("true")
					}
					return exec.Command("false")
				}
				return exec.Command(name, arg...)
			}
			defer func() {
				execCommand = exec.Command
			}()

			// 実行
			CreateSession(tt.sessionName)

			// ログ出力の検証
			switch tt.wantLogLevel {
			case "info":
				if !containsMessage(mockLog.infoMessages, tt.wantLogMessage) {
					t.Errorf("期待するログが出力されませんでした: %s", tt.wantLogMessage)
				}
			case "error":
				if !containsMessage(mockLog.errorMessages, tt.wantLogMessage) {
					t.Errorf("期待するエラーログが出力されませんでした: %s", tt.wantLogMessage)
				}
			}
		})
	}
}

// containsMessage はメッセージリストに指定された部分文字列を含むメッセージがあるかチェック
func containsMessage(messages []string, substr string) bool {
	for _, msg := range messages {
		if strings.Contains(msg, substr) {
			return true
		}
	}
	return false
}
