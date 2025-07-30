package cmd

import (
	"os"
	"path/filepath"
	"strings"
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
	}{
		{
			name: "デフォルト設定（環境変数なし）",
			setupEnv: func() {
				os.Unsetenv("XDG_CONFIG_HOME")
			},
			cleanupEnv: func() {},
			wantContains: []string{
				".config/osoba/osoba.yml",
				".config/osoba/osoba.yaml",
				".osoba.yml",
				".osoba.yaml",
			},
		},
		{
			name: "XDG_CONFIG_HOMEが設定されている場合",
			setupEnv: func() {
				os.Setenv("XDG_CONFIG_HOME", "/custom/config")
			},
			cleanupEnv: func() {
				os.Unsetenv("XDG_CONFIG_HOME")
			},
			wantContains: []string{
				"/custom/config/osoba/osoba.yml",
				"/custom/config/osoba/osoba.yaml",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			paths := getConfigFilePaths()

			for _, want := range tt.wantContains {
				found := false
				for _, path := range paths {
					// パスの最後の部分が一致するかチェック
					if strings.HasSuffix(filepath.ToSlash(path), filepath.ToSlash(want)) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected paths to contain %s, but it was not found in %v", want, paths)
				}
			}
		})
	}
}

func TestFindConfigFile(t *testing.T) {
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
		name       string
		setupFiles func()
		setupEnv   func()
		wantFound  bool
		wantPath   string
	}{
		{
			name:       "設定ファイルが存在しない場合",
			setupFiles: func() {},
			setupEnv: func() {
				os.Unsetenv("XDG_CONFIG_HOME")
			},
			wantFound: false,
			wantPath:  "",
		},
		{
			name: "デフォルトパスに設定ファイルが存在する場合",
			setupFiles: func() {
				configDir := filepath.Join(tmpDir, ".config", "osoba")
				os.MkdirAll(configDir, 0755)
				configFile := filepath.Join(configDir, "osoba.yml")
				os.WriteFile(configFile, []byte("test"), 0644)
			},
			setupEnv: func() {
				os.Unsetenv("XDG_CONFIG_HOME")
			},
			wantFound: true,
			wantPath:  filepath.Join(tmpDir, ".config", "osoba", "osoba.yml"),
		},
		{
			name: "XDG_CONFIG_HOMEのパスに設定ファイルが存在する場合",
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
			wantFound: true,
			wantPath:  filepath.Join(tmpDir, "xdg", "osoba", "osoba.yml"),
		},
		{
			name: "ホームディレクトリに.osoba.ymlが存在する場合",
			setupFiles: func() {
				configFile := filepath.Join(tmpDir, ".osoba.yml")
				os.WriteFile(configFile, []byte("test"), 0644)
			},
			setupEnv: func() {
				os.Unsetenv("XDG_CONFIG_HOME")
			},
			wantFound: true,
			wantPath:  filepath.Join(tmpDir, ".osoba.yml"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// クリーンアップ
			os.RemoveAll(filepath.Join(tmpDir, ".config"))
			os.RemoveAll(filepath.Join(tmpDir, "xdg"))
			os.Remove(filepath.Join(tmpDir, ".osoba.yml"))
			os.Remove(filepath.Join(tmpDir, ".osoba.yaml"))

			tt.setupFiles()
			tt.setupEnv()

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
