package watcher

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestLoadConfig(t *testing.T) {
	// テスト用の一時ファイルを作成
	configContent := `
github:
  token: "test-token"
  owner: "douhashi"
  repo: "osoba"
  poll_interval: 10s
  max_retries: 5
  retry_base_delay: 2s
  labels:
    - "status:needs-plan"
    - "status:ready"
    - "status:review-requested"
`

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	t.Run("設定ファイルを正しく読み込める", func(t *testing.T) {
		config, err := LoadConfig(tmpfile.Name())
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if config.GitHub.Token != "test-token" {
			t.Errorf("Token = %v, want %v", config.GitHub.Token, "test-token")
		}
		if config.GitHub.Owner != "douhashi" {
			t.Errorf("Owner = %v, want %v", config.GitHub.Owner, "douhashi")
		}
		if config.GitHub.Repo != "osoba" {
			t.Errorf("Repo = %v, want %v", config.GitHub.Repo, "osoba")
		}
		if config.GitHub.PollInterval != 10*time.Second {
			t.Errorf("PollInterval = %v, want %v", config.GitHub.PollInterval, 10*time.Second)
		}
		if config.GitHub.MaxRetries != 5 {
			t.Errorf("MaxRetries = %v, want %v", config.GitHub.MaxRetries, 5)
		}
		if config.GitHub.RetryBaseDelay != 2*time.Second {
			t.Errorf("RetryBaseDelay = %v, want %v", config.GitHub.RetryBaseDelay, 2*time.Second)
		}
		if len(config.GitHub.Labels) != 3 {
			t.Errorf("Labels length = %v, want %v", len(config.GitHub.Labels), 3)
		}
	})

	t.Run("存在しないファイルはエラー", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/file.yaml")
		if err == nil {
			t.Error("LoadConfig() should return error for nonexistent file")
		}
	})
}

func TestLoadConfigWithDefaults(t *testing.T) {
	// 最小限の設定ファイル
	configContent := `
github:
  token: "test-token"
  owner: "douhashi"
  repo: "osoba"
`

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	t.Run("デフォルト値が適用される", func(t *testing.T) {
		config, err := LoadConfig(tmpfile.Name())
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		// デフォルト値の確認
		if config.GitHub.PollInterval != 5*time.Second {
			t.Errorf("PollInterval = %v, want %v (default)", config.GitHub.PollInterval, 5*time.Second)
		}
		if config.GitHub.MaxRetries != 3 {
			t.Errorf("MaxRetries = %v, want %v (default)", config.GitHub.MaxRetries, 3)
		}
		if config.GitHub.RetryBaseDelay != time.Second {
			t.Errorf("RetryBaseDelay = %v, want %v (default)", config.GitHub.RetryBaseDelay, time.Second)
		}
	})
}

func TestConfigFromEnv(t *testing.T) {
	// 環境変数を設定
	os.Setenv("OSOBA_GITHUB_TOKEN", "env-token")
	os.Setenv("OSOBA_GITHUB_OWNER", "env-owner")
	os.Setenv("OSOBA_GITHUB_REPO", "env-repo")
	defer func() {
		os.Unsetenv("OSOBA_GITHUB_TOKEN")
		os.Unsetenv("OSOBA_GITHUB_OWNER")
		os.Unsetenv("OSOBA_GITHUB_REPO")
	}()

	// 最小限の設定ファイル
	configContent := `
github:
  token: ""
  owner: ""
  repo: ""
`

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	t.Run("環境変数から設定を読み込める", func(t *testing.T) {
		config, err := LoadConfig(tmpfile.Name())
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if config.GitHub.Token != "env-token" {
			t.Errorf("Token = %v, want %v", config.GitHub.Token, "env-token")
		}
		if config.GitHub.Owner != "env-owner" {
			t.Errorf("Owner = %v, want %v", config.GitHub.Owner, "env-owner")
		}
		if config.GitHub.Repo != "env-repo" {
			t.Errorf("Repo = %v, want %v", config.GitHub.Repo, "env-repo")
		}
	})
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "正常な設定",
			config: &Config{
				GitHub: GitHubConfig{
					Token:          "test-token",
					Owner:          "douhashi",
					Repo:           "osoba",
					PollInterval:   5 * time.Second,
					MaxRetries:     3,
					RetryBaseDelay: time.Second,
					Labels:         []string{"status:ready"},
				},
			},
			wantErr: false,
		},
		{
			name: "トークンが空",
			config: &Config{
				GitHub: GitHubConfig{
					Token:          "",
					Owner:          "douhashi",
					Repo:           "osoba",
					PollInterval:   5 * time.Second,
					MaxRetries:     3,
					RetryBaseDelay: time.Second,
					Labels:         []string{"status:ready"},
				},
			},
			wantErr: true,
			errMsg:  "github.token is required",
		},
		{
			name: "オーナーが空",
			config: &Config{
				GitHub: GitHubConfig{
					Token:          "test-token",
					Owner:          "",
					Repo:           "osoba",
					PollInterval:   5 * time.Second,
					MaxRetries:     3,
					RetryBaseDelay: time.Second,
					Labels:         []string{"status:ready"},
				},
			},
			wantErr: true,
			errMsg:  "github.owner is required",
		},
		{
			name: "リポジトリが空",
			config: &Config{
				GitHub: GitHubConfig{
					Token:          "test-token",
					Owner:          "douhashi",
					Repo:           "",
					PollInterval:   5 * time.Second,
					MaxRetries:     3,
					RetryBaseDelay: time.Second,
					Labels:         []string{"status:ready"},
				},
			},
			wantErr: true,
			errMsg:  "github.repo is required",
		},
		{
			name: "ポーリング間隔が無効",
			config: &Config{
				GitHub: GitHubConfig{
					Token:          "test-token",
					Owner:          "douhashi",
					Repo:           "osoba",
					PollInterval:   500 * time.Millisecond,
					MaxRetries:     3,
					RetryBaseDelay: time.Second,
					Labels:         []string{"status:ready"},
				},
			},
			wantErr: true,
			errMsg:  "github.poll_interval must be at least 1 second",
		},
		{
			name: "最大リトライ回数が無効",
			config: &Config{
				GitHub: GitHubConfig{
					Token:          "test-token",
					Owner:          "douhashi",
					Repo:           "osoba",
					PollInterval:   5 * time.Second,
					MaxRetries:     0,
					RetryBaseDelay: time.Second,
					Labels:         []string{"status:ready"},
				},
			},
			wantErr: true,
			errMsg:  "github.max_retries must be at least 1",
		},
		{
			name: "リトライ基本遅延が無効",
			config: &Config{
				GitHub: GitHubConfig{
					Token:          "test-token",
					Owner:          "douhashi",
					Repo:           "osoba",
					PollInterval:   5 * time.Second,
					MaxRetries:     3,
					RetryBaseDelay: 0,
					Labels:         []string{"status:ready"},
				},
			},
			wantErr: true,
			errMsg:  "github.retry_base_delay must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("ValidateConfig() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestGetViperDefaults(t *testing.T) {
	v := viper.New()
	SetViperDefaults(v)

	t.Run("デフォルト値が設定される", func(t *testing.T) {
		if v.GetDuration("github.poll_interval") != 5*time.Second {
			t.Errorf("poll_interval = %v, want %v", v.GetDuration("github.poll_interval"), 5*time.Second)
		}
		if v.GetInt("github.max_retries") != 3 {
			t.Errorf("max_retries = %v, want %v", v.GetInt("github.max_retries"), 3)
		}
		if v.GetDuration("github.retry_base_delay") != time.Second {
			t.Errorf("retry_base_delay = %v, want %v", v.GetDuration("github.retry_base_delay"), time.Second)
		}
	})
}
