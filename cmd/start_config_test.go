package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/douhashi/osoba/internal/testutil/helpers"
)

func TestCheckConfigFileExists(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir := t.TempDir()

	// 元の環境変数を保存
	origHome := os.Getenv("HOME")
	origXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")

	// テスト用の環境変数を設定
	os.Setenv("HOME", tmpDir)
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("XDG_CONFIG_HOME", origXDGConfigHome)
	}()

	tests := []struct {
		name            string
		setupFiles      func()
		setupEnv        func()
		mockStat        func(*helpers.FunctionMocker)
		wantErr         bool
		wantErrContains string
	}{
		{
			name: "設定ファイルが存在する場合",
			setupFiles: func() {
				configDir := filepath.Join(tmpDir, ".config", "osoba")
				os.MkdirAll(configDir, 0755)
				configFile := filepath.Join(configDir, "osoba.yml")
				os.WriteFile(configFile, []byte("test"), 0644)
			},
			setupEnv: func() {
				os.Unsetenv("XDG_CONFIG_HOME")
			},
			wantErr: false,
		},
		{
			name: "設定ファイルが存在しない場合",
			setupFiles: func() {
				// 何もしない（ファイルを作成しない）
			},
			setupEnv: func() {
				os.Unsetenv("XDG_CONFIG_HOME")
			},
			wantErr:         true,
			wantErrContains: "設定ファイルが見つかりません",
		},
		{
			name: "XDG_CONFIG_HOMEに設定ファイルが存在する場合",
			setupFiles: func() {
				xdgDir := filepath.Join(tmpDir, "xdg")
				configDir := filepath.Join(xdgDir, "osoba")
				os.MkdirAll(configDir, 0755)
				configFile := filepath.Join(configDir, "osoba.yml")
				os.WriteFile(configFile, []byte("test"), 0644)
			},
			setupEnv: func() {
				os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "xdg"))
			},
			wantErr: false,
		},
		{
			name:       "ホームディレクトリの取得に失敗した場合",
			setupFiles: func() {},
			setupEnv: func() {
				os.Unsetenv("XDG_CONFIG_HOME")
			},
			mockStat: func(mocker *helpers.FunctionMocker) {
				// os.UserHomeDirをモックして失敗させる
				mocker.MockFunc(&osUserHomeDirFunc, func() (string, error) {
					return "", errors.New("home directory not found")
				})
			},
			wantErr:         true,
			wantErrContains: "設定ファイルが見つかりません",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// クリーンアップ
			os.RemoveAll(filepath.Join(tmpDir, ".config"))
			os.RemoveAll(filepath.Join(tmpDir, "xdg"))
			os.Remove(filepath.Join(tmpDir, ".osoba.yml"))
			os.Remove(filepath.Join(tmpDir, ".osoba.yaml"))

			// モッカーのセットアップ
			mocker := helpers.NewFunctionMocker()
			defer mocker.Restore()

			tt.setupFiles()
			tt.setupEnv()

			if tt.mockStat != nil {
				tt.mockStat(mocker)
			}

			errBuf := new(bytes.Buffer)
			err := checkConfigFileExists(errBuf)

			if (err != nil) != tt.wantErr {
				t.Errorf("checkConfigFileExists() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErrContains != "" {
				errOutput := errBuf.String()
				if !bytes.Contains([]byte(errOutput), []byte(tt.wantErrContains)) {
					t.Errorf("checkConfigFileExists() error output = %v, want to contain %v", errOutput, tt.wantErrContains)
				}
			}
		})
	}
}
