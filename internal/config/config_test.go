package config

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/claude"
)

func TestNewConfig(t *testing.T) {
	t.Run("正常系: デフォルト設定でConfigを作成できる", func(t *testing.T) {
		cfg := NewConfig()
		if cfg == nil {
			t.Fatal("NewConfig() returned nil")
		}

		// デフォルト値の確認
		if cfg.GitHub.PollInterval != 20*time.Second {
			t.Errorf("default poll interval = %v, want 20s", cfg.GitHub.PollInterval)
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

		// auto_plan_issueのデフォルト値確認
		if cfg.GitHub.AutoPlanIssue != false {
			t.Errorf("default auto_plan_issue = %v, want false", cfg.GitHub.AutoPlanIssue)
		}

		// すべてのフェーズで --dangerously-skip-permissions が設定されていることを確認
		phases := []string{"plan", "implement", "review"}
		for _, phase := range phases {
			if phaseConfig, exists := cfg.Claude.Phases[phase]; exists {
				found := false
				for _, arg := range phaseConfig.Args {
					if arg == "--dangerously-skip-permissions" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Claude %s phase should have --dangerously-skip-permissions", phase)
				}
			} else {
				t.Errorf("Claude %s phase not found", phase)
			}
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
  poll_interval: 10s
  auto_plan_issue: true
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
				if cfg.GitHub.PollInterval != 10*time.Second {
					t.Errorf("poll interval = %v, want 10s", cfg.GitHub.PollInterval)
				}
				if cfg.GitHub.AutoPlanIssue != true {
					t.Errorf("auto_plan_issue = %v, want true", cfg.GitHub.AutoPlanIssue)
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
  poll_interval: 10s
`,
			envVars: map[string]string{
				"GITHUB_TOKEN": "env-token",
			},
			wantErr: false,
			checkFunc: func(cfg *Config, t *testing.T) {
			},
		},
		{
			name:       "正常系: ghコマンドからトークンを取得",
			configFile: "test_config_gh_token.yml",
			configContent: `
github:
  poll_interval: 5s
`,
			envVars: map[string]string{},
			wantErr: false,
			checkFunc: func(cfg *Config, t *testing.T) {
				// ghコマンドが使える環境では、トークンが自動取得される
				// テスト環境では具体的な値は検証しない
				// Token が空でないことだけ確認する方が安全（CI環境での動作を考慮）
			},
		},
		{
			name:       "異常系: 不正なYAMLファイル",
			configFile: "test_config_invalid.yml",
			configContent: `
github:
  poll_interval: [invalid yaml
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 既存の環境変数をバックアップしてクリア
			envBackup := make(map[string]string)
			for _, key := range []string{"GITHUB_TOKEN", "OSOBA_GITHUB_TOKEN"} {
				if val, exists := os.LookupEnv(key); exists {
					envBackup[key] = val
				}
				os.Unsetenv(key)
			}
			defer func() {
				// 環境変数を復元
				for key, val := range envBackup {
					os.Setenv(key, val)
				}
				for _, key := range []string{"GITHUB_TOKEN", "OSOBA_GITHUB_TOKEN"} {
					if _, exists := envBackup[key]; !exists {
						os.Unsetenv(key)
					}
				}
			}()

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
			name: "異常系: ポーリング間隔が短すぎる",
			cfg: &Config{
				GitHub: GitHubConfig{
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
				Plan:            "status:needs-plan",
				Ready:           "status:ready",
				Review:          "status:review-requested",
				RequiresChanges: "status:requires-changes",
				Revising:        "status:revising",
			},
		},
	}

	labels := cfg.GetLabels()

	expected := []string{"status:needs-plan", "status:ready", "status:review-requested", "status:requires-changes", "status:revising"}
	if len(labels) != len(expected) {
		t.Fatalf("GetLabels() returned %d labels, want %d", len(labels), len(expected))
	}

	for i, label := range labels {
		if label != expected[i] {
			t.Errorf("GetLabels()[%d] = %v, want %v", i, label, expected[i])
		}
	}
}

func TestConfig_AutoMergeLGTM(t *testing.T) {
	t.Run("正常系: デフォルト値がtrueであることを確認", func(t *testing.T) {
		cfg := NewConfig()
		if cfg == nil {
			t.Fatal("NewConfig() returned nil")
		}

		// デフォルト値がtrueであることを確認
		if !cfg.GitHub.AutoMergeLGTM {
			t.Errorf("default AutoMergeLGTM = %v, want true", cfg.GitHub.AutoMergeLGTM)
		}
	})

	t.Run("正常系: 設定ファイルでfalseに設定した場合の読み込み", func(t *testing.T) {
		// テスト用の設定ファイルを作成
		content := `
github:
  poll_interval: 10s
  auto_merge_lgtm: false
`
		err := os.WriteFile("test_auto_merge_false.yml", []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test config file: %v", err)
		}
		defer os.Remove("test_auto_merge_false.yml")

		cfg := NewConfig()
		err = cfg.Load("test_auto_merge_false.yml")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// falseが読み込まれることを確認
		if cfg.GitHub.AutoMergeLGTM {
			t.Errorf("AutoMergeLGTM = %v, want false", cfg.GitHub.AutoMergeLGTM)
		}
	})

	t.Run("正常系: 設定ファイルでtrueに設定した場合の読み込み", func(t *testing.T) {
		// テスト用の設定ファイルを作成
		content := `
github:
  poll_interval: 10s
  auto_merge_lgtm: true
`
		err := os.WriteFile("test_auto_merge_true.yml", []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test config file: %v", err)
		}
		defer os.Remove("test_auto_merge_true.yml")

		cfg := NewConfig()
		err = cfg.Load("test_auto_merge_true.yml")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// trueが読み込まれることを確認
		if !cfg.GitHub.AutoMergeLGTM {
			t.Errorf("AutoMergeLGTM = %v, want true", cfg.GitHub.AutoMergeLGTM)
		}
	})

	t.Run("正常系: 設定ファイルに項目がない場合はデフォルト値を使用", func(t *testing.T) {
		// テスト用の設定ファイルを作成（auto_merge_lgtmフィールドなし）
		content := `
github:
  poll_interval: 10s
`
		err := os.WriteFile("test_auto_merge_default.yml", []byte(content), 0644)
		if err != nil {
			t.Fatalf("failed to create test config file: %v", err)
		}
		defer os.Remove("test_auto_merge_default.yml")

		cfg := NewConfig()
		err = cfg.Load("test_auto_merge_default.yml")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// デフォルト値（true）が使用されることを確認
		if !cfg.GitHub.AutoMergeLGTM {
			t.Errorf("AutoMergeLGTM = %v, want true (default)", cfg.GitHub.AutoMergeLGTM)
		}
	})
}

func TestConfig_LoadOrDefault(t *testing.T) {
	t.Run("正常系: ファイルが存在しない場合はデフォルト値を使う", func(t *testing.T) {
		cfg := NewConfig()
		actualPath := cfg.LoadOrDefault("non_existent_file.yml")

		// パスが空文字列であることを確認
		if actualPath != "" {
			t.Errorf("actualPath = %v, want empty string", actualPath)
		}

		// デフォルト値が設定されていることを確認
		if cfg.GitHub.PollInterval != 20*time.Second {
			t.Errorf("poll interval = %v, want 20s", cfg.GitHub.PollInterval)
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
		actualPath := cfg.LoadOrDefault("test_load_or_default.yml")

		// 実際のパスが返されることを確認
		if actualPath != "test_load_or_default.yml" {
			t.Errorf("actualPath = %v, want test_load_or_default.yml", actualPath)
		}

		if cfg.GitHub.PollInterval != 15*time.Second {
			t.Errorf("poll interval = %v, want 15s", cfg.GitHub.PollInterval)
		}
	})
}

// TestConfigSettingsReflection は設定値が各コンポーネントに正しく反映されることを確認する
func TestConfigSettingsReflection(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		checkFunc     func(*Config, *testing.T)
	}{
		{
			name: "GitHub設定値の反映確認",
			configContent: `
github:
  poll_interval: 15s
  labels:
    plan: "status:needs-planning"
    ready: "status:implementation-ready"
    review: "status:review-needed"
tmux:
  session_prefix: "custom-osoba-"
claude:
  phases:
    plan:
      args: ["--custom-arg", "--verbose"]
      prompt: "/custom:plan {{issue-number}}"
    implement:
      args: ["--skip-permissions", "--force"]
      prompt: "/custom:implement {{issue-number}}"
    review:
      args: ["--review-mode"]
      prompt: "/custom:review {{issue-number}}"
`,
			checkFunc: func(cfg *Config, t *testing.T) {
				// GitHub設定の確認
				expectedPollInterval := 15 * time.Second
				if cfg.GitHub.PollInterval != expectedPollInterval {
					t.Errorf("GitHub.PollInterval = %v, want %v", cfg.GitHub.PollInterval, expectedPollInterval)
				}

				if cfg.GitHub.Labels.Plan != "status:needs-planning" {
					t.Errorf("GitHub.Labels.Plan = %v, want status:needs-planning", cfg.GitHub.Labels.Plan)
				}

				if cfg.GitHub.Labels.Ready != "status:implementation-ready" {
					t.Errorf("GitHub.Labels.Ready = %v, want status:implementation-ready", cfg.GitHub.Labels.Ready)
				}

				if cfg.GitHub.Labels.Review != "status:review-needed" {
					t.Errorf("GitHub.Labels.Review = %v, want status:review-needed", cfg.GitHub.Labels.Review)
				}

				// tmux設定の確認
				if cfg.Tmux.SessionPrefix != "custom-osoba-" {
					t.Errorf("Tmux.SessionPrefix = %v, want custom-osoba-", cfg.Tmux.SessionPrefix)
				}

				// Claude設定の確認
				if cfg.Claude == nil {
					t.Fatal("Claude config is nil")
				}

				// Plan フェーズの確認
				if planPhase, exists := cfg.Claude.Phases["plan"]; exists {
					expectedArgs := []string{"--custom-arg", "--verbose"}
					if len(planPhase.Args) != len(expectedArgs) {
						t.Errorf("Claude plan args length = %d, want %d", len(planPhase.Args), len(expectedArgs))
					} else {
						for i, arg := range expectedArgs {
							if planPhase.Args[i] != arg {
								t.Errorf("Claude plan args[%d] = %v, want %v", i, planPhase.Args[i], arg)
							}
						}
					}

					expectedPrompt := "/custom:plan {{issue-number}}"
					if planPhase.Prompt != expectedPrompt {
						t.Errorf("Claude plan prompt = %v, want %v", planPhase.Prompt, expectedPrompt)
					}
				} else {
					t.Error("Claude plan phase not found")
				}

				// Implement フェーズの確認
				if implPhase, exists := cfg.Claude.Phases["implement"]; exists {
					expectedArgs := []string{"--skip-permissions", "--force"}
					if len(implPhase.Args) != len(expectedArgs) {
						t.Errorf("Claude implement args length = %d, want %d", len(implPhase.Args), len(expectedArgs))
					} else {
						for i, arg := range expectedArgs {
							if implPhase.Args[i] != arg {
								t.Errorf("Claude implement args[%d] = %v, want %v", i, implPhase.Args[i], arg)
							}
						}
					}

					expectedPrompt := "/custom:implement {{issue-number}}"
					if implPhase.Prompt != expectedPrompt {
						t.Errorf("Claude implement prompt = %v, want %v", implPhase.Prompt, expectedPrompt)
					}
				} else {
					t.Error("Claude implement phase not found")
				}

				// Review フェーズの確認
				if reviewPhase, exists := cfg.Claude.Phases["review"]; exists {
					expectedArgs := []string{"--review-mode"}
					if len(reviewPhase.Args) != len(expectedArgs) {
						t.Errorf("Claude review args length = %d, want %d", len(reviewPhase.Args), len(expectedArgs))
					} else {
						for i, arg := range expectedArgs {
							if reviewPhase.Args[i] != arg {
								t.Errorf("Claude review args[%d] = %v, want %v", i, reviewPhase.Args[i], arg)
							}
						}
					}

					expectedPrompt := "/custom:review {{issue-number}}"
					if reviewPhase.Prompt != expectedPrompt {
						t.Errorf("Claude review prompt = %v, want %v", reviewPhase.Prompt, expectedPrompt)
					}
				} else {
					t.Error("Claude review phase not found")
				}
			},
		},
		{
			name: "環境変数展開の確認",
			configContent: `
github:
  poll_interval: 10s
`,
			checkFunc: func(cfg *Config, t *testing.T) {
				// 環境変数が設定されている場合のテスト（このテストでは実際の値の確認は行わない）
				// 設定読み込み自体が成功していることを確認
				if cfg.GitHub.PollInterval != 10*time.Second {
					t.Errorf("GitHub.PollInterval = %v, want 10s", cfg.GitHub.PollInterval)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テスト用設定ファイルを作成
			filename := "test_config_reflection_" + tt.name + ".yml"
			err := os.WriteFile(filename, []byte(tt.configContent), 0644)
			if err != nil {
				t.Fatalf("設定ファイル作成に失敗: %v", err)
			}
			defer os.Remove(filename)

			// 設定を読み込み
			cfg := NewConfig()
			if err := cfg.Load(filename); err != nil {
				t.Fatalf("設定読み込みに失敗: %v", err)
			}

			// テスト関数を実行
			tt.checkFunc(cfg, t)
		})
	}
}

// TestConfig_ValidateClaudeConfig はClaude設定のバリデーションをテストする
func TestConfig_ValidateClaudeConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		wantErr     bool
		errContains string
	}{
		{
			name: "正常系: 全てのフェーズが適切に設定されている",
			config: &Config{
				Claude: &claude.ClaudeConfig{
					Phases: map[string]*claude.PhaseConfig{
						"plan": {
							Args:   []string{"--arg1"},
							Prompt: "/osoba:plan {{issue-number}}",
						},
						"implement": {
							Args:   []string{"--arg2"},
							Prompt: "/osoba:implement {{issue-number}}",
						},
						"review": {
							Args:   []string{"--arg3"},
							Prompt: "/osoba:review {{issue-number}}",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "異常系: planフェーズが不足",
			config: &Config{
				Claude: &claude.ClaudeConfig{
					Phases: map[string]*claude.PhaseConfig{
						"implement": {
							Prompt: "/osoba:implement {{issue-number}}",
						},
						"review": {
							Prompt: "/osoba:review {{issue-number}}",
						},
					},
				},
			},
			wantErr:     true,
			errContains: "required phase 'plan' is missing",
		},
		{
			name: "異常系: プロンプトが空",
			config: &Config{
				Claude: &claude.ClaudeConfig{
					Phases: map[string]*claude.PhaseConfig{
						"plan": {
							Prompt: "",
						},
						"implement": {
							Prompt: "/osoba:implement {{issue-number}}",
						},
						"review": {
							Prompt: "/osoba:review {{issue-number}}",
						},
					},
				},
			},
			wantErr:     true,
			errContains: "phase 'plan' prompt is empty",
		},
		{
			name: "異常系: テンプレート変数が不足",
			config: &Config{
				Claude: &claude.ClaudeConfig{
					Phases: map[string]*claude.PhaseConfig{
						"plan": {
							Prompt: "/osoba:plan",
						},
						"implement": {
							Prompt: "/osoba:implement {{issue-number}}",
						},
						"review": {
							Prompt: "/osoba:review {{issue-number}}",
						},
					},
				},
			},
			wantErr:     true,
			errContains: "phase 'plan' prompt must contain {{issue-number}} template variable",
		},
		{
			name: "正常系: Claude設定がnil",
			config: &Config{
				Claude: nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validateClaudeConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateClaudeConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !findSubstring(err.Error(), tt.errContains) {
					t.Errorf("validateClaudeConfig() error = %v, want error containing %v", err, tt.errContains)
				}
			}
		})
	}
}

// TestGetGitHubToken はGitHubトークン取得をテストする
func TestGetGitHubToken(t *testing.T) {
	tests := []struct {
		name        string
		ghAuthToken string
		ghCmdExists bool
		want        string
		wantSource  string
	}{
		{
			name:        "ghコマンドのトークンが取得できる",
			ghAuthToken: "gh-auth-token",
			ghCmdExists: true,
			want:        "gh-auth-token",
			wantSource:  "gh auth token",
		},
		{
			name:        "ghコマンドが存在しない場合",
			ghAuthToken: "",
			ghCmdExists: false,
			want:        "",
			wantSource:  "",
		},
		{
			name:        "ghコマンドがエラーを返す場合",
			ghAuthToken: "", // 空文字列はエラーをシミュレート
			ghCmdExists: true,
			want:        "",
			wantSource:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ghコマンドのモック
			originalGhAuthTokenFunc := GhAuthTokenFunc
			GhAuthTokenFunc = func() (string, error) {
				if !tt.ghCmdExists {
					return "", fmt.Errorf("gh command not found")
				}
				if tt.ghAuthToken == "" {
					return "", fmt.Errorf("gh auth token failed")
				}
				return tt.ghAuthToken, nil
			}
			defer func() {
				GhAuthTokenFunc = originalGhAuthTokenFunc
			}()

			// Configを作成
			cfg := NewConfig()

			// トークンを取得
			token, source := GetGitHubToken(cfg)

			// 結果を検証
			if token != tt.want {
				t.Errorf("GetGitHubToken() token = %v, want %v", token, tt.want)
			}
			if source != tt.wantSource {
				t.Errorf("GetGitHubToken() source = %v, want %v", source, tt.wantSource)
			}
		})
	}
}

// TestLogLevelConfig はログレベル設定のテストを行う
func TestLogLevelConfig(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		envVars       map[string]string
		wantLogLevel  string
		wantLogFormat string
	}{
		{
			name: "設定ファイルでログレベルを指定",
			configContent: `
log:
  level: debug
  format: json
`,
			envVars:       map[string]string{},
			wantLogLevel:  "debug",
			wantLogFormat: "json",
		},
		{
			name: "環境変数でログレベルを指定",
			configContent: `
log:
  level: info
  format: text
`,
			envVars: map[string]string{
				"OSOBA_LOG_LEVEL": "warn",
			},
			wantLogLevel:  "warn",
			wantLogFormat: "text",
		},
		{
			name: "デフォルト値でログレベルを設定",
			configContent: `
github:
  poll_interval: 10s
`,
			envVars:       map[string]string{},
			wantLogLevel:  "info",
			wantLogFormat: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// テスト用設定ファイルを作成
			filename := "test_log_level_" + tt.name + ".yml"
			err := os.WriteFile(filename, []byte(tt.configContent), 0644)
			if err != nil {
				t.Fatalf("設定ファイル作成に失敗: %v", err)
			}
			defer os.Remove(filename)

			// 環境変数のバックアップとクリア
			envBackup := make(map[string]string)
			envKeys := []string{"OSOBA_LOG_LEVEL", "OSOBA_LOG_FORMAT"}
			for _, key := range envKeys {
				if val, exists := os.LookupEnv(key); exists {
					envBackup[key] = val
				}
				os.Unsetenv(key)
			}
			defer func() {
				// 環境変数を復元
				for key, val := range envBackup {
					os.Setenv(key, val)
				}
			}()

			// テスト用の環境変数を設定
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			// 設定を読み込み
			cfg := NewConfig()
			if err := cfg.Load(filename); err != nil {
				t.Fatalf("設定読み込みに失敗: %v", err)
			}

			// ログレベルの確認
			if cfg.Log.Level != tt.wantLogLevel {
				t.Errorf("Log.Level = %v, want %v", cfg.Log.Level, tt.wantLogLevel)
			}

			// ログフォーマットの確認
			if cfg.Log.Format != tt.wantLogFormat {
				t.Errorf("Log.Format = %v, want %v", cfg.Log.Format, tt.wantLogFormat)
			}
		})
	}
}

// TestLogConfig_CreateLogger はログ設定から実際のロガーを作成するテストを行う
func TestLogConfig_CreateLogger(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		logFormat string
		wantErr   bool
	}{
		{
			name:      "正常系: debug/text",
			logLevel:  "debug",
			logFormat: "text",
			wantErr:   false,
		},
		{
			name:      "正常系: info/json",
			logLevel:  "info",
			logFormat: "json",
			wantErr:   false,
		},
		{
			name:      "正常系: warn/text",
			logLevel:  "warn",
			logFormat: "text",
			wantErr:   false,
		},
		{
			name:      "正常系: error/json",
			logLevel:  "error",
			logFormat: "json",
			wantErr:   false,
		},
		{
			name:      "異常系: 無効なログレベル",
			logLevel:  "invalid",
			logFormat: "text",
			wantErr:   true,
		},
		{
			name:      "異常系: 無効なフォーマット",
			logLevel:  "info",
			logFormat: "invalid",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Log: LogConfig{
					Level:  tt.logLevel,
					Format: tt.logFormat,
				},
			}

			logger, err := cfg.CreateLogger()
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && logger == nil {
				t.Error("CreateLogger() returned nil logger")
			}
		})
	}
}

// TestAutoPlanIssueConfig は auto_plan_issue 設定に関するテスト
func TestAutoPlanIssueConfig(t *testing.T) {
	t.Run("正常系: auto_plan_issueがtrueに設定される", func(t *testing.T) {
		configContent := `
github:
  poll_interval: 5s
  auto_plan_issue: true
  auto_merge_lgtm: false
tmux:
  session_prefix: "osoba-"
`
		configFile := "test_auto_plan_issue.yml"
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("failed to create test config file: %v", err)
		}
		defer os.Remove(configFile)

		cfg := NewConfig()
		err = cfg.Load(configFile)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.GitHub.AutoPlanIssue != true {
			t.Errorf("auto_plan_issue = %v, want true", cfg.GitHub.AutoPlanIssue)
		}
		if cfg.GitHub.AutoMergeLGTM != false {
			t.Errorf("auto_merge_lgtm = %v, want false", cfg.GitHub.AutoMergeLGTM)
		}
	})

	t.Run("正常系: auto_plan_issueがfalseに設定される", func(t *testing.T) {
		configContent := `
github:
  poll_interval: 5s
  auto_plan_issue: false
  auto_merge_lgtm: true
`
		configFile := "test_auto_plan_issue_false.yml"
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("failed to create test config file: %v", err)
		}
		defer os.Remove(configFile)

		cfg := NewConfig()
		err = cfg.Load(configFile)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.GitHub.AutoPlanIssue != false {
			t.Errorf("auto_plan_issue = %v, want false", cfg.GitHub.AutoPlanIssue)
		}
		if cfg.GitHub.AutoMergeLGTM != true {
			t.Errorf("auto_merge_lgtm = %v, want true", cfg.GitHub.AutoMergeLGTM)
		}
	})

	t.Run("正常系: auto_plan_issue未設定時はデフォルトfalse", func(t *testing.T) {
		configContent := `
github:
  poll_interval: 5s
  auto_merge_lgtm: true
`
		configFile := "test_auto_plan_issue_default.yml"
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("failed to create test config file: %v", err)
		}
		defer os.Remove(configFile)

		cfg := NewConfig()
		err = cfg.Load(configFile)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.GitHub.AutoPlanIssue != false {
			t.Errorf("auto_plan_issue = %v, want false (default)", cfg.GitHub.AutoPlanIssue)
		}
	})
}
