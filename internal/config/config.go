package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/spf13/viper"
)

// Config はアプリケーション全体の設定
type Config struct {
	GitHub GitHubConfig         `mapstructure:"github"`
	Tmux   TmuxConfig           `mapstructure:"tmux"`
	Claude *claude.ClaudeConfig `mapstructure:"claude"`
}

// GitHubConfig はGitHub関連の設定
type GitHubConfig struct {
	Token        string             `mapstructure:"token"`
	PollInterval time.Duration      `mapstructure:"poll_interval"`
	Labels       LabelConfig        `mapstructure:"labels"`
	Messages     PhaseMessageConfig `mapstructure:"messages"`
}

// LabelConfig は監視対象のラベル設定
type LabelConfig struct {
	Plan   string `mapstructure:"plan"`
	Ready  string `mapstructure:"ready"`
	Review string `mapstructure:"review"`
}

// PhaseMessageConfig はフェーズ開始時のコメントメッセージ設定
type PhaseMessageConfig struct {
	Plan      string `mapstructure:"plan"`
	Implement string `mapstructure:"implement"`
	Review    string `mapstructure:"review"`
}

// TmuxConfig はtmux関連の設定
type TmuxConfig struct {
	SessionPrefix string `mapstructure:"session_prefix"`
}

// NewDefaultPhaseMessageConfig はデフォルトのフェーズメッセージ設定を返す
func NewDefaultPhaseMessageConfig() PhaseMessageConfig {
	return PhaseMessageConfig{
		Plan:      "osoba: 計画を作成します",
		Implement: "osoba: 実装を開始します",
		Review:    "osoba: レビューを開始します",
	}
}

// NewConfig は新しいConfigを作成する
func NewConfig() *Config {
	return &Config{
		GitHub: GitHubConfig{
			PollInterval: 5 * time.Second,
			Labels: LabelConfig{
				Plan:   "status:needs-plan",
				Ready:  "status:ready",
				Review: "status:review-requested",
			},
			Messages: NewDefaultPhaseMessageConfig(),
		},
		Tmux: TmuxConfig{
			SessionPrefix: "osoba-",
		},
		Claude: claude.NewDefaultClaudeConfig(),
	}
}

// Load は設定ファイルから設定を読み込む
func (c *Config) Load(configPath string) error {
	v := viper.New()

	// 設定ファイルのパスを設定
	v.SetConfigFile(configPath)

	// 環境変数の設定
	v.SetEnvPrefix("OSOBA")
	v.AutomaticEnv()

	// OSOBA_GITHUB_TOKENを優先、GITHUB_TOKENもサポート
	v.BindEnv("github.token", "OSOBA_GITHUB_TOKEN", "GITHUB_TOKEN")

	// デフォルト値の設定
	v.SetDefault("github.poll_interval", 5*time.Second)
	v.SetDefault("github.labels.plan", "status:needs-plan")
	v.SetDefault("github.labels.ready", "status:ready")
	v.SetDefault("github.labels.review", "status:review-requested")
	v.SetDefault("github.messages.plan", "osoba: 計画を作成します")
	v.SetDefault("github.messages.implement", "osoba: 実装を開始します")
	v.SetDefault("github.messages.review", "osoba: レビューを開始します")
	v.SetDefault("tmux.session_prefix", "osoba-")

	// Claude設定のデフォルト値
	v.SetDefault("claude.phases.plan.args", []string{"--dangerously-skip-permissions"})
	v.SetDefault("claude.phases.plan.prompt", "/osoba:plan {{issue-number}}")
	v.SetDefault("claude.phases.implement.args", []string{"--dangerously-skip-permissions"})
	v.SetDefault("claude.phases.implement.prompt", "/osoba:implement {{issue-number}}")
	v.SetDefault("claude.phases.review.args", []string{"--dangerously-skip-permissions"})
	v.SetDefault("claude.phases.review.prompt", "/osoba:review {{issue-number}}")

	// 設定ファイルを読み込む
	if err := v.ReadInConfig(); err != nil {
		return err
	}

	// 設定を構造体にマッピング
	if err := v.Unmarshal(c); err != nil {
		return err
	}

	return nil
}

// LoadOrDefault は設定ファイルを読み込み、失敗した場合はデフォルト値を使用する
func (c *Config) LoadOrDefault(configPath string) {
	// ファイルが存在しない場合はデフォルト値を使用
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Claudeのデフォルト設定を確保
		if c.Claude == nil {
			c.Claude = claude.NewDefaultClaudeConfig()
		}
		return
	}

	// 設定ファイルを読み込む（エラーは無視）
	_ = c.Load(configPath)

	// Claudeのデフォルト設定を確保
	if c.Claude == nil {
		c.Claude = claude.NewDefaultClaudeConfig()
	}
}

// Validate は設定の妥当性を検証する
func (c *Config) Validate() error {
	if c.GitHub.Token == "" {
		return errors.New("GitHub token is required")
	}

	if c.GitHub.PollInterval < 1*time.Second {
		return errors.New("poll interval must be at least 1 second")
	}

	// ラベルが空の場合はデフォルト値を設定
	if c.GitHub.Labels.Plan == "" {
		c.GitHub.Labels.Plan = "status:needs-plan"
	}
	if c.GitHub.Labels.Ready == "" {
		c.GitHub.Labels.Ready = "status:ready"
	}
	if c.GitHub.Labels.Review == "" {
		c.GitHub.Labels.Review = "status:review-requested"
	}

	// tmux設定のバリデーション
	if c.Tmux.SessionPrefix == "" {
		c.Tmux.SessionPrefix = "osoba-"
	}

	// Claude設定のバリデーション
	if c.Claude != nil {
		if err := c.validateClaudeConfig(); err != nil {
			return fmt.Errorf("invalid claude config: %w", err)
		}
	}

	return nil
}

// validateClaudeConfig はClaude設定の妥当性を検証する
func (c *Config) validateClaudeConfig() error {
	if c.Claude == nil {
		return nil
	}

	// 必須フェーズの確認
	requiredPhases := []string{"plan", "implement", "review"}
	for _, phase := range requiredPhases {
		if _, exists := c.Claude.Phases[phase]; !exists {
			return fmt.Errorf("required phase '%s' is missing", phase)
		}
	}

	// 各フェーズの設定を検証
	for phase, config := range c.Claude.Phases {
		if config == nil {
			return fmt.Errorf("phase '%s' config is nil", phase)
		}

		// プロンプトが空でないことを確認
		if config.Prompt == "" {
			return fmt.Errorf("phase '%s' prompt is empty", phase)
		}

		// プロンプトに必要なテンプレート変数が含まれているかチェック
		if phase == "plan" || phase == "implement" || phase == "review" {
			if !containsTemplate(config.Prompt, "{{issue-number}}") {
				return fmt.Errorf("phase '%s' prompt must contain {{issue-number}} template variable", phase)
			}
		}
	}

	return nil
}

// containsTemplate はテンプレート変数が含まれているかチェックする
func containsTemplate(prompt, template string) bool {
	return len(prompt) > 0 && len(template) > 0 &&
		(prompt == template ||
			(len(prompt) >= len(template) &&
				findSubstring(prompt, template)))
}

// findSubstring は部分文字列を検索する
func findSubstring(str, substr string) bool {
	if len(substr) > len(str) {
		return false
	}
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetLabels は監視対象のラベルをスライスで返す
func (c *Config) GetLabels() []string {
	return []string{
		c.GitHub.Labels.Plan,
		c.GitHub.Labels.Ready,
		c.GitHub.Labels.Review,
	}
}

// GetPhaseMessage は指定されたフェーズのメッセージを返す
func (c *Config) GetPhaseMessage(phase string) (string, bool) {
	switch phase {
	case "plan":
		return c.GitHub.Messages.Plan, true
	case "implement":
		return c.GitHub.Messages.Implement, true
	case "review":
		return c.GitHub.Messages.Review, true
	default:
		return "", false
	}
}
