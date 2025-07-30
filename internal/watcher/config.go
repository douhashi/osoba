package watcher

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config はwatcherの設定を表す
type Config struct {
	GitHub GitHubConfig `mapstructure:"github"`
}

// GitHubConfig はGitHub関連の設定を表す
type GitHubConfig struct {
	Owner          string        `mapstructure:"owner"`
	Repo           string        `mapstructure:"repo"`
	PollInterval   time.Duration `mapstructure:"poll_interval"`
	MaxRetries     int           `mapstructure:"max_retries"`
	RetryBaseDelay time.Duration `mapstructure:"retry_base_delay"`
	Labels         []string      `mapstructure:"labels"`
}

// LoadConfig は設定ファイルを読み込む
func LoadConfig(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	// デフォルト値を設定
	SetViperDefaults(v)

	// 環境変数を読み込む
	v.SetEnvPrefix("OSOBA")
	v.AutomaticEnv()
	// 環境変数のマッピングを明示的に設定
	v.BindEnv("github.owner", "OSOBA_GITHUB_OWNER")
	v.BindEnv("github.repo", "OSOBA_GITHUB_REPO")

	// 設定ファイルを読み込む
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 設定を構造体にマッピング
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// ghコマンドを使用するため、トークンの取得は不要

	// 設定の検証
	if err := ValidateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// SetViperDefaults はViperのデフォルト値を設定する
func SetViperDefaults(v *viper.Viper) {
	v.SetDefault("github.poll_interval", 5*time.Second)
	v.SetDefault("github.max_retries", 3)
	v.SetDefault("github.retry_base_delay", time.Second)
}

// ValidateConfig は設定の妥当性を検証する
func ValidateConfig(cfg *Config) error {
	if cfg.GitHub.Owner == "" {
		return errors.New("github.owner is required")
	}
	if cfg.GitHub.Repo == "" {
		return errors.New("github.repo is required")
	}
	if cfg.GitHub.PollInterval < time.Second {
		return errors.New("github.poll_interval must be at least 1 second")
	}
	if cfg.GitHub.MaxRetries < 1 {
		return errors.New("github.max_retries must be at least 1")
	}
	if cfg.GitHub.RetryBaseDelay <= 0 {
		return errors.New("github.retry_base_delay must be positive")
	}
	return nil
}

// ValidatePollInterval はポーリング間隔の妥当性を検証する
func ValidatePollInterval(interval time.Duration) error {
	if interval < time.Second {
		return errors.New("poll interval must be at least 1 second")
	}
	return nil
}
