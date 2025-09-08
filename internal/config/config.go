package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/logger"
	"github.com/spf13/viper"
)

// Config はアプリケーション全体の設定
type Config struct {
	GitHub     GitHubConfig         `mapstructure:"github"`
	Tmux       TmuxConfig           `mapstructure:"tmux"`
	Claude     *claude.ClaudeConfig `mapstructure:"claude"`
	Log        LogConfig            `mapstructure:"log"`
	IsTestMode bool                 // テストモードかどうかを示すフラグ
}

// GitHubConfig はGitHub関連の設定
type GitHubConfig struct {
	PollInterval   time.Duration      `mapstructure:"poll_interval"`
	PRPollInterval time.Duration      `mapstructure:"pr_poll_interval"` // PR監視専用のポーリング間隔
	Labels         LabelConfig        `mapstructure:"labels"`
	Messages       PhaseMessageConfig `mapstructure:"messages"`
	AutoMergeLGTM  bool               `mapstructure:"auto_merge_lgtm"` // status:lgtmラベルが付いたPRを自動マージする機能の有効/無効
	AutoPlanIssue  bool               `mapstructure:"auto_plan_issue"` // 処理中のIssueがない場合に自動的に次のIssueをplanフェーズに移行させる機能の有効/無効
	AutoRevisePR   bool               `mapstructure:"auto_revise_pr"`  // status:requires-changesラベルが付いたPRに対して自動的にreviseアクションを実行する機能の有効/無効
}

// LabelConfig は監視対象のラベル設定
type LabelConfig struct {
	Plan            string `mapstructure:"plan"`
	Ready           string `mapstructure:"ready"`
	Review          string `mapstructure:"review"`
	RequiresChanges string `mapstructure:"requires_changes"`
	Revising        string `mapstructure:"revising"`
}

// PhaseMessageConfig はフェーズ開始時のコメントメッセージ設定
type PhaseMessageConfig struct {
	Plan      string `mapstructure:"plan"`
	Implement string `mapstructure:"implement"`
	Review    string `mapstructure:"review"`
}

// TmuxConfig はtmux関連の設定
type TmuxConfig struct {
	SessionPrefix     string `mapstructure:"session_prefix"`
	MaxPanesPerWindow int    `mapstructure:"max_panes_per_window"`
	LimitPanesEnabled bool   `mapstructure:"limit_panes_enabled"`
}

// LogConfig はログ関連の設定
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
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
	// テストモードかどうかを判定
	isTestMode := os.Getenv("OSOBA_TEST_MODE") == "true"

	// セッションプレフィックスをテストモードに応じて設定
	sessionPrefix := "osoba-"
	if isTestMode {
		sessionPrefix = "test-osoba-"
	}

	return &Config{
		GitHub: GitHubConfig{
			PollInterval:   20 * time.Second,
			PRPollInterval: 20 * time.Second, // PR監視間隔もデフォルト20秒
			Labels: LabelConfig{
				Plan:            "status:needs-plan",
				Ready:           "status:ready",
				Review:          "status:review-requested",
				RequiresChanges: "status:requires-changes",
				Revising:        "status:revising",
			},
			Messages:      NewDefaultPhaseMessageConfig(),
			AutoMergeLGTM: true,  // デフォルトで自動マージ機能を有効化
			AutoPlanIssue: false, // デフォルトで自動計画機能を無効化
			AutoRevisePR:  true,  // デフォルトで自動Revise機能を有効化
		},
		Tmux: TmuxConfig{
			SessionPrefix:     sessionPrefix,
			MaxPanesPerWindow: 3,
			LimitPanesEnabled: true,
		},
		Claude: claude.NewDefaultClaudeConfig(),
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
		IsTestMode: isTestMode,
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

	// 環境変数のバインドは不要（ghコマンドのみ使用）

	// ログレベルの環境変数バインド
	v.BindEnv("log.level", "OSOBA_LOG_LEVEL")
	v.BindEnv("log.format", "OSOBA_LOG_FORMAT")

	// デフォルト値の設定
	v.SetDefault("github.poll_interval", 20*time.Second)
	v.SetDefault("github.pr_poll_interval", 20*time.Second) // PR監視間隔のデフォルト値
	v.SetDefault("github.labels.plan", "status:needs-plan")
	v.SetDefault("github.labels.ready", "status:ready")
	v.SetDefault("github.labels.review", "status:review-requested")
	v.SetDefault("github.labels.requires_changes", "status:requires-changes")
	v.SetDefault("github.labels.revising", "status:revising")
	v.SetDefault("github.messages.plan", "osoba: 計画を作成します")
	v.SetDefault("github.messages.implement", "osoba: 実装を開始します")
	v.SetDefault("github.messages.review", "osoba: レビューを開始します")
	v.SetDefault("github.auto_merge_lgtm", true)
	v.SetDefault("github.auto_plan_issue", false)
	v.SetDefault("github.auto_revise_pr", true)
	v.SetDefault("tmux.session_prefix", "osoba-")

	// ログ設定のデフォルト値
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")

	// Claude設定のデフォルト値
	v.SetDefault("claude.phases.plan.args", []string{"--dangerously-skip-permissions"})
	v.SetDefault("claude.phases.plan.prompt", "/osoba:plan {{issue-number}}")
	v.SetDefault("claude.phases.implement.args", []string{"--dangerously-skip-permissions"})
	v.SetDefault("claude.phases.implement.prompt", "/osoba:implement {{issue-number}}")
	v.SetDefault("claude.phases.review.args", []string{"--dangerously-skip-permissions"})
	v.SetDefault("claude.phases.review.prompt", "/osoba:review {{issue-number}}")
	v.SetDefault("claude.phases.revise.args", []string{"--dangerously-skip-permissions"})
	v.SetDefault("claude.phases.revise.prompt", "/osoba:revise {{issue-number}}")

	// 設定ファイルを読み込む
	if err := v.ReadInConfig(); err != nil {
		return err
	}

	// 設定を構造体にマッピング
	if err := v.Unmarshal(c); err != nil {
		return err
	}

	// テストモードの場合、セッションプレフィックスを上書き
	if os.Getenv("OSOBA_TEST_MODE") == "true" {
		c.IsTestMode = true
		c.Tmux.SessionPrefix = "test-osoba-"
	}

	// ghコマンドを使用するため、トークンの取得は不要

	return nil
}

// LoadOrDefault は設定ファイルを読み込み、失敗した場合はデフォルト値を使用する
// 実際に読み込んだファイルパスを返す（読み込まなかった場合は空文字列）
func (c *Config) LoadOrDefault(configPath string) string {
	actualPath := configPath

	// configPathが空の場合はカレントディレクトリのデフォルトパスを試す
	if configPath == "" {
		// カレントディレクトリのパスの候補を試す
		defaultPaths := []string{
			".osoba.yml",
			".osoba.yaml",
		}

		for _, path := range defaultPaths {
			if _, err := os.Stat(path); err == nil {
				actualPath = path
				break
			}
		}
	}

	// 設定ファイルが見つかった場合は読み込む
	if actualPath != "" {
		if _, err := os.Stat(actualPath); err == nil {
			// 設定ファイルを読み込む（エラーは無視）
			if err := c.Load(actualPath); err == nil {
				// 読み込み成功時のみパスを返す
				return actualPath
			}
		}
	}

	// Claudeのデフォルト設定を確保
	if c.Claude == nil {
		c.Claude = claude.NewDefaultClaudeConfig()
	}

	// テストモードの場合、セッションプレフィックスを上書き
	if os.Getenv("OSOBA_TEST_MODE") == "true" {
		c.IsTestMode = true
		c.Tmux.SessionPrefix = "test-osoba-"
	}

	return ""
}

// Validate は設定の妥当性を検証する
func (c *Config) Validate() error {
	if c.GitHub.PollInterval < 1*time.Second {
		return errors.New("poll interval must be at least 1 second")
	}
	// PRPollIntervalが未設定（0）の場合はPollIntervalと同じ値を使用
	if c.GitHub.PRPollInterval == 0 {
		c.GitHub.PRPollInterval = c.GitHub.PollInterval
	}
	if c.GitHub.PRPollInterval < 1*time.Second {
		return errors.New("PR poll interval must be at least 1 second")
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
	if c.GitHub.Labels.RequiresChanges == "" {
		c.GitHub.Labels.RequiresChanges = "status:requires-changes"
	}
	if c.GitHub.Labels.Revising == "" {
		c.GitHub.Labels.Revising = "status:revising"
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
		c.GitHub.Labels.RequiresChanges,
		c.GitHub.Labels.Revising,
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

// GhAuthTokenFunc はテスト用のモック可能な関数変数（公開）
var GhAuthTokenFunc = executeGhAuthToken

// executeGhAuthToken は実際の gh auth token コマンドを実行する
func executeGhAuthToken() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetGitHubToken はGitHubトークンを取得し、取得元を返す
// ghコマンドのみをサポート
func GetGitHubToken(cfg *Config) (token string, source string) {
	// gh auth tokenコマンドを試す
	if ghToken, err := GhAuthTokenFunc(); err == nil && ghToken != "" {
		return ghToken, "gh auth token"
	}

	return "", ""
}

// SetDefaults は空フィールドにデフォルト値を設定する
func (c *Config) SetDefaults() {
	if c.GitHub.Labels.Plan == "" {
		c.GitHub.Labels.Plan = "status:needs-plan"
	}
	if c.GitHub.Labels.Ready == "" {
		c.GitHub.Labels.Ready = "status:ready"
	}
	if c.GitHub.Labels.Review == "" {
		c.GitHub.Labels.Review = "status:review-requested"
	}
	if c.GitHub.Labels.RequiresChanges == "" {
		c.GitHub.Labels.RequiresChanges = "status:requires-changes"
	}
	if c.GitHub.Labels.Revising == "" {
		c.GitHub.Labels.Revising = "status:revising"
	}
}

// CreateLogger はログ設定からロガーを作成する
func (c *Config) CreateLogger() (logger.Logger, error) {
	return logger.New(
		logger.WithLevel(c.Log.Level),
		logger.WithFormat(c.Log.Format),
	)
}
