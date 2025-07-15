package config

import (
	"os"
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	t.Run("正常系: デフォルト設定でConfigを作成できる", func(t *testing.T) {
		cfg := NewConfig()
		if cfg == nil {
			t.Fatal("NewConfig() returned nil")
		}

		// デフォルト値の確認
		if cfg.GitHub.PollInterval != 5*time.Second {
			t.Errorf("default poll interval = %v, want 5s", cfg.GitHub.PollInterval)
		}
		if cfg.Tmux.SessionPrefix != "osoba-" {
			t.Errorf("default session prefix = %v, want osoba-", cfg.Tmux.SessionPrefix)
		}
		// Claude設定のデフォルト値確認
		if cfg.Claude == nil {
			t.Error("Claude config is nil")
		}
		if cfg.Claude.Phases == nil {
			t.Error("Claude phases is nil")
		}
		if _, exists := cfg.Claude.Phases["plan"]; !exists {
			t.Error("Claude plan phase not found")
		}
	})
}

func TestConfig_Load(t *testing.T) {
	tests := []struct {
		name          string
		configFile    string
		configContent string
		envVars       map[string]string
		wantErr       bool
		checkFunc     func(*Config, *testing.T)
	}{
		{
			name:       "正常系: YAMLファイルから設定を読み込める",
			configFile: "test_config.yml",
			configContent: `
github:
  token: test-token-from-file
  poll_interval: 10s
  labels:
    plan: "status:needs-plan"
    ready: "status:ready"
    review: "status:review-requested"
tmux:
  session_prefix: "test-osoba-"
claude:
  phases:
    plan:
      args: ["--skip-confirmation"]
      prompt: "/osoba:plan {{issue-number}}"
    implement:
      args: []
      prompt: "/osoba:implement {{issue-number}}"
`,
			wantErr: false,
			checkFunc: func(cfg *Config, t *testing.T) {
				if cfg.GitHub.Token != "test-token-from-file" {
					t.Errorf("token = %v, want test-token-from-file", cfg.GitHub.Token)
				}
				if cfg.GitHub.PollInterval != 10*time.Second {
					t.Errorf("poll interval = %v, want 10s", cfg.GitHub.PollInterval)
				}
				if cfg.GitHub.Labels.Plan != "status:needs-plan" {
					t.Errorf("plan label = %v, want status:needs-plan", cfg.GitHub.Labels.Plan)
				}
				if cfg.Tmux.SessionPrefix != "test-osoba-" {
					t.Errorf("session prefix = %v, want test-osoba-", cfg.Tmux.SessionPrefix)
				}
				// Claude設定の確認
				if cfg.Claude == nil || cfg.Claude.Phases == nil {
					t.Error("Claude config not loaded properly")
				}
				if planPhase, exists := cfg.Claude.Phases["plan"]; exists {
					if len(planPhase.Args) != 1 || planPhase.Args[0] != "--skip-confirmation" {
						t.Errorf("Claude plan args = %v, want [--skip-confirmation]", planPhase.Args)
					}
				} else {
					t.Error("Claude plan phase not found")
				}
			},
		},
		{
			name:       "正常系: 環境変数が設定ファイルより優先される",
			configFile: "test_config_env.yml",
			configContent: `
github:
  token: file-token
  poll_interval: 10s
`,
			envVars: map[string]string{
				"OSOBA_GITHUB_TOKEN": "env-token",
			},
			wantErr: false,
			checkFunc: func(cfg *Config, t *testing.T) {
				if cfg.GitHub.Token != "env-token" {
					t.Errorf("token = %v, want env-token", cfg.GitHub.Token)
				}
			},
		},
		{
			name:       "正常系: 環境変数GITHUB_TOKENも使える",
			configFile: "test_config_github_token.yml",
			configContent: `
github:
  poll_interval: 5s
`,
			envVars: map[string]string{
				"GITHUB_TOKEN": "github-env-token",
			},
			wantErr: false,
			checkFunc: func(cfg *Config, t *testing.T) {
				if cfg.GitHub.Token != "github-env-token" {
					t.Errorf("token = %v, want github-env-token", cfg.GitHub.Token)
				}
			},
		},
		{
			name:       "異常系: 不正なYAMLファイル",
			configFile: "test_config_invalid.yml",
			configContent: `
github:
  token: [invalid yaml
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テスト用の設定ファイルを作成
			if tt.configContent != "" {
				err := os.WriteFile(tt.configFile, []byte(tt.configContent), 0644)
				if err != nil {
					t.Fatalf("failed to create test config file: %v", err)
				}
				defer os.Remove(tt.configFile)
			}

			// 環境変数を設定
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			cfg := NewConfig()
			err := cfg.Load(tt.configFile)

			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(cfg, t)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "正常系: 有効な設定",
			cfg: &Config{
				GitHub: GitHubConfig{
					Token:        "test-token",
					PollInterval: 5 * time.Second,
					Labels: LabelConfig{
						Plan:   "status:needs-plan",
						Ready:  "status:ready",
						Review: "status:review-requested",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "異常系: GitHubトークンが空",
			cfg: &Config{
				GitHub: GitHubConfig{
					Token:        "",
					PollInterval: 5 * time.Second,
				},
			},
			wantErr: true,
			errMsg:  "GitHub token is required",
		},
		{
			name: "異常系: ポーリング間隔が短すぎる",
			cfg: &Config{
				GitHub: GitHubConfig{
					Token:        "test-token",
					PollInterval: 500 * time.Millisecond,
				},
			},
			wantErr: true,
			errMsg:  "poll interval must be at least 1 second",
		},
		{
			name: "正常系: ラベルが空でもデフォルト値が使われる",
			cfg: &Config{
				GitHub: GitHubConfig{
					Token:        "test-token",
					PollInterval: 5 * time.Second,
					Labels:       LabelConfig{},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestConfig_GetLabels(t *testing.T) {
	cfg := &Config{
		GitHub: GitHubConfig{
			Labels: LabelConfig{
				Plan:   "status:needs-plan",
				Ready:  "status:ready",
				Review: "status:review-requested",
			},
		},
	}

	labels := cfg.GetLabels()

	expected := []string{"status:needs-plan", "status:ready", "status:review-requested"}
	if len(labels) != len(expected) {
		t.Fatalf("GetLabels() returned %d labels, want %d", len(labels), len(expected))
	}

	for i, label := range labels {
		if label != expected[i] {
			t.Errorf("GetLabels()[%d] = %v, want %v", i, label, expected[i])
		}
	}
}

func TestConfig_LoadOrDefault(t *testing.T) {
	t.Run("正常系: ファイルが存在しない場合はデフォルト値を使う", func(t *testing.T) {
		cfg := NewConfig()
		cfg.LoadOrDefault("non_existent_file.yml")

		// デフォルト値が設定されていることを確認
		if cfg.GitHub.PollInterval != 5*time.Second {
			t.Errorf("poll interval = %v, want 5s", cfg.GitHub.PollInterval)
		}
	})

	t.Run("正常系: ファイルが存在する場合は読み込む", func(t *testing.T) {
		// テスト用の設定ファイルを作成
		content := `
github:
  poll_interval: 15s
`
		err := os.WriteFile("test_load_or_default.yml", []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test config file: %v", err)
		}
		defer os.Remove("test_load_or_default.yml")

		cfg := NewConfig()
		cfg.LoadOrDefault("test_load_or_default.yml")

		if cfg.GitHub.PollInterval != 15*time.Second {
			t.Errorf("poll interval = %v, want 15s", cfg.GitHub.PollInterval)
		}
	})
}
