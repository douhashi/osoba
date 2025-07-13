package config

import (
	"errors"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config はアプリケーション全体の設定
type Config struct {
	GitHub GitHubConfig `mapstructure:"github"`
	Tmux   TmuxConfig   `mapstructure:"tmux"`
}

// GitHubConfig はGitHub関連の設定
type GitHubConfig struct {
	Token        string        `mapstructure:"token"`
	PollInterval time.Duration `mapstructure:"poll_interval"`
	Labels       LabelConfig   `mapstructure:"labels"`
}

// LabelConfig は監視対象のラベル設定
type LabelConfig struct {
	Plan   string `mapstructure:"plan"`
	Ready  string `mapstructure:"ready"`
	Review string `mapstructure:"review"`
}

// TmuxConfig はtmux関連の設定
type TmuxConfig struct {
	SessionPrefix string `mapstructure:"session_prefix"`
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
		},
		Tmux: TmuxConfig{
			SessionPrefix: "osoba-",
		},
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

	// GITHUB_TOKENもサポート
	v.BindEnv("github.token", "GITHUB_TOKEN", "OSOBA_GITHUB_TOKEN")

	// デフォルト値の設定
	v.SetDefault("github.poll_interval", 5*time.Second)
	v.SetDefault("github.labels.plan", "status:needs-plan")
	v.SetDefault("github.labels.ready", "status:ready")
	v.SetDefault("github.labels.review", "status:review-requested")
	v.SetDefault("tmux.session_prefix", "osoba-")

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
		return
	}

	// 設定ファイルを読み込む（エラーは無視）
	_ = c.Load(configPath)
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

	return nil
}

// GetLabels は監視対象のラベルをスライスで返す
func (c *Config) GetLabels() []string {
	return []string{
		c.GitHub.Labels.Plan,
		c.GitHub.Labels.Ready,
		c.GitHub.Labels.Review,
	}
}
