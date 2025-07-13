package tmux

import (
	"errors"
	"os/exec"
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
