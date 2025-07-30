package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestSetupConfigFile(t *testing.T) {
	// モック関数を保存
	origStatFunc := statFunc
	origWriteFileFunc := writeFileFunc
	defer func() {
		statFunc = origStatFunc
		writeFileFunc = origWriteFileFunc
	}()

	tests := []struct {
		name           string
		setupMocks     func()
		wantErr        bool
		wantConfigPath string
		wantContent    string
	}{
		{
			name: "正常系: 新規設定ファイルを作成",
			setupMocks: func() {
				// ファイルが存在しない
				statFunc = func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				}
				// ファイル書き込み成功
				writeFileFunc = func(name string, data []byte, perm os.FileMode) error {
					// カレントディレクトリの.osoba.ymlに書き込まれることを確認
					if name != ".osoba.yml" {
						t.Errorf("unexpected file path: got %s, want .osoba.yml", name)
					}
					// 内容を確認
					content := string(data)
					if !strings.Contains(content, "github:") {
						t.Error("config content should contain 'github:' section")
					}
					if !strings.Contains(content, "tmux:") {
						t.Error("config content should contain 'tmux:' section")
					}
					if !strings.Contains(content, "claude:") {
						t.Error("config content should contain 'claude:' section")
					}
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "正常系: 既存ファイルがある場合はスキップ",
			setupMocks: func() {
				// ファイルが既に存在
				statFunc = func(name string) (os.FileInfo, error) {
					return nil, nil // 存在する
				}
				// writeFileが呼ばれないことを確認
				writeFileFunc = func(name string, data []byte, perm os.FileMode) error {
					t.Error("writeFile should not be called when file exists")
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "異常系: ファイル書き込みエラー",
			setupMocks: func() {
				// ファイルが存在しない
				statFunc = func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				}
				// ファイル書き込み失敗
				writeFileFunc = func(name string, data []byte, perm os.FileMode) error {
					return os.ErrPermission
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			out := &bytes.Buffer{}
			err := setupConfigFile(out)

			if (err != nil) != tt.wantErr {
				t.Errorf("setupConfigFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSetupConfigFile_NoGlobalDirectory(t *testing.T) {
	// モック関数を保存
	origStatFunc := statFunc
	origWriteFileFunc := writeFileFunc
	origMkdirAllFunc := mkdirAllFunc
	defer func() {
		statFunc = origStatFunc
		writeFileFunc = origWriteFileFunc
		mkdirAllFunc = origMkdirAllFunc
	}()

	// mkdirAllが呼ばれないことを確認
	mkdirAllFunc = func(path string, perm os.FileMode) error {
		if strings.Contains(path, ".config/osoba") {
			t.Error("mkdirAll should not be called for global config directory")
		}
		return nil
	}

	// ファイルが存在しない
	statFunc = func(name string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}

	// ファイル書き込み成功
	writeFileFunc = func(name string, data []byte, perm os.FileMode) error {
		return nil
	}

	out := &bytes.Buffer{}
	err := setupConfigFile(out)
	if err != nil {
		t.Errorf("setupConfigFile() unexpected error: %v", err)
	}
}
