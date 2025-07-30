package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfig_LoadOrDefault_CurrentDirectory(t *testing.T) {
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

	t.Run("カレントディレクトリの.osoba.ymlを読み込む", func(t *testing.T) {
		// カレントディレクトリに設定ファイルを作成
		content := `
github:
  poll_interval: 30s
`
		err := os.WriteFile(".osoba.yml", []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test config file: %v", err)
		}
		defer os.Remove(".osoba.yml")

		cfg := NewConfig()
		actualPath := cfg.LoadOrDefault("")

		// カレントディレクトリの設定ファイルが読み込まれたことを確認
		if actualPath != ".osoba.yml" {
			t.Errorf("actualPath = %v, want .osoba.yml", actualPath)
		}

		// 設定内容が反映されていることを確認
		if cfg.GitHub.PollInterval != 30*time.Second {
			t.Errorf("poll interval = %v, want 30s", cfg.GitHub.PollInterval)
		}
	})

	t.Run("カレントディレクトリの.osoba.yamlを読み込む", func(t *testing.T) {
		// カレントディレクトリに設定ファイルを作成（.yaml拡張子）
		content := `
github:
  poll_interval: 45s
`
		err := os.WriteFile(".osoba.yaml", []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test config file: %v", err)
		}
		defer os.Remove(".osoba.yaml")

		cfg := NewConfig()
		actualPath := cfg.LoadOrDefault("")

		// カレントディレクトリの設定ファイルが読み込まれたことを確認
		if actualPath != ".osoba.yaml" {
			t.Errorf("actualPath = %v, want .osoba.yaml", actualPath)
		}

		// 設定内容が反映されていることを確認
		if cfg.GitHub.PollInterval != 45*time.Second {
			t.Errorf("poll interval = %v, want 45s", cfg.GitHub.PollInterval)
		}
	})

	t.Run(".osoba.ymlが優先される", func(t *testing.T) {
		// 両方のファイルを作成
		contentYml := `
github:
  poll_interval: 10s
`
		contentYaml := `
github:
  poll_interval: 20s
`
		err := os.WriteFile(".osoba.yml", []byte(contentYml), 0644)
		if err != nil {
			t.Fatalf("failed to create .osoba.yml: %v", err)
		}
		defer os.Remove(".osoba.yml")

		err = os.WriteFile(".osoba.yaml", []byte(contentYaml), 0644)
		if err != nil {
			t.Fatalf("failed to create .osoba.yaml: %v", err)
		}
		defer os.Remove(".osoba.yaml")

		cfg := NewConfig()
		actualPath := cfg.LoadOrDefault("")

		// .osoba.ymlが優先されることを確認
		if actualPath != ".osoba.yml" {
			t.Errorf("actualPath = %v, want .osoba.yml", actualPath)
		}

		// .osoba.ymlの内容が反映されていることを確認
		if cfg.GitHub.PollInterval != 10*time.Second {
			t.Errorf("poll interval = %v, want 10s", cfg.GitHub.PollInterval)
		}
	})

	t.Run("グローバル設定ディレクトリは無視される", func(t *testing.T) {
		// ホームディレクトリに設定ファイルを作成
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("Cannot get home directory")
		}

		// グローバル設定ファイルのパス
		globalConfigDir := filepath.Join(home, ".config", "osoba")
		globalConfigPath := filepath.Join(globalConfigDir, "osoba.yml")

		// グローバル設定が存在してもカレントディレクトリを優先することを確認
		cfg := NewConfig()
		actualPath := cfg.LoadOrDefault("")

		// グローバル設定ファイルではなく、空文字列が返ることを確認
		if actualPath == globalConfigPath {
			t.Errorf("should not load global config file: %v", actualPath)
		}

		// Claude設定がデフォルトで初期化されていることを確認
		if cfg.Claude == nil {
			t.Error("Claude config should be initialized with default values")
		}
	})
}

func TestConfig_LoadOrDefault_NoGlobalPaths(t *testing.T) {
	// グローバルパスを含まないことを確認するためのテスト
	cfg := NewConfig()
	actualPath := cfg.LoadOrDefault("")

	// グローバルパスが返されないことを確認
	if actualPath != "" && filepath.IsAbs(actualPath) && !isCurrentDirPath(actualPath) {
		t.Errorf("LoadOrDefault should not return global path: %v", actualPath)
	}
}

func isCurrentDirPath(path string) bool {
	// パスがカレントディレクトリの相対パスかどうかを確認
	return path == ".osoba.yml" || path == ".osoba.yaml"
}
