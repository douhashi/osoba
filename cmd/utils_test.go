package cmd

import (
	"os"
	"testing"
)

func TestGetConfigFilePaths(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tmpDir := t.TempDir()

	// 元のHOME環境変数を保存
	origHome := os.Getenv("HOME")
	defer func() { os.Setenv("HOME", origHome) }()

	// HOMEをテスト用ディレクトリに設定
	os.Setenv("HOME", tmpDir)

	tests := []struct {
		name         string
		setupEnv     func()
		cleanupEnv   func()
		wantContains []string
		wantLength   int
	}{
		{
			name: "カレントディレクトリのみを返す",
			setupEnv: func() {
				os.Unsetenv("XDG_CONFIG_HOME")
			},
			cleanupEnv: func() {},
			wantContains: []string{
				".osoba.yml",
				".osoba.yaml",
			},
			wantLength: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			paths := getConfigFilePaths()

			// 長さの確認
			if len(paths) != tt.wantLength {
				t.Errorf("Expected %d paths, but got %d", tt.wantLength, len(paths))
			}

			// 内容の確認
			for i, want := range tt.wantContains {
				if i >= len(paths) {
					break
				}
				if paths[i] != want {
					t.Errorf("Expected path[%d] to be %s, but got %s", i, want, paths[i])
				}
			}
		})
	}
}

func TestFindConfigFile(t *testing.T) {
	// 元のワーキングディレクトリを保存
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origWd)

	// テスト用の一時ディレクトリを作成し、そこに移動
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		setupFiles func()
		wantFound  bool
		wantPath   string
	}{
		{
			name:       "設定ファイルが存在しない場合",
			setupFiles: func() {},
			wantFound:  false,
			wantPath:   "",
		},
		{
			name: "カレントディレクトリに.osoba.ymlが存在する場合",
			setupFiles: func() {
				os.WriteFile(".osoba.yml", []byte("test"), 0644)
			},
			wantFound: true,
			wantPath:  ".osoba.yml",
		},
		{
			name: "カレントディレクトリに.osoba.yamlが存在する場合",
			setupFiles: func() {
				os.WriteFile(".osoba.yaml", []byte("test"), 0644)
			},
			wantFound: true,
			wantPath:  ".osoba.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// クリーンアップ
			os.Remove(".osoba.yml")
			os.Remove(".osoba.yaml")

			tt.setupFiles()

			path, found := findConfigFile()

			if found != tt.wantFound {
				t.Errorf("findConfigFile() found = %v, want %v", found, tt.wantFound)
			}

			if found && path != tt.wantPath {
				t.Errorf("findConfigFile() path = %v, want %v", path, tt.wantPath)
			}
		})
	}
}
