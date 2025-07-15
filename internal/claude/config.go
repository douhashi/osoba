package claude

// PhaseConfig はフェーズごとのClaude実行設定
type PhaseConfig struct {
	Args   []string `mapstructure:"args"`
	Prompt string   `mapstructure:"prompt"`
}

// ClaudeConfig はClaude実行の全体設定
type ClaudeConfig struct {
	Phases map[string]*PhaseConfig `mapstructure:"phases"`
}

// NewDefaultClaudeConfig はデフォルトのClaude設定を生成する
func NewDefaultClaudeConfig() *ClaudeConfig {
	return &ClaudeConfig{
		Phases: map[string]*PhaseConfig{
			"plan": {
				Args:   []string{"--dangerously-skip-permissions"},
				Prompt: "/osoba:plan {{issue-number}}",
			},
			"implement": {
				Args:   []string{"--dangerously-skip-permissions"},
				Prompt: "/osoba:implement {{issue-number}}",
			},
			"review": {
				Args:   []string{"--dangerously-skip-permissions"},
				Prompt: "/osoba:review {{issue-number}}",
			},
		},
	}
}

// GetPhase は指定されたフェーズの設定を取得する
func (c *ClaudeConfig) GetPhase(phase string) (*PhaseConfig, bool) {
	config, exists := c.Phases[phase]
	return config, exists
}
