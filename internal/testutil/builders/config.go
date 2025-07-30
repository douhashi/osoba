package builders

import (
	"time"

	"github.com/douhashi/osoba/internal/claude"
	"github.com/douhashi/osoba/internal/config"
	"github.com/douhashi/osoba/internal/github"
)

// ConfigBuilder builds config.Config instances for testing
type ConfigBuilder struct {
	cfg *config.Config
}

// NewConfigBuilder creates a new ConfigBuilder with sensible defaults
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		cfg: &config.Config{
			GitHub: config.GitHubConfig{
				PollInterval: 5 * time.Minute,
				Labels: config.LabelConfig{
					Plan:   "status:needs-plan",
					Ready:  "status:ready",
					Review: "status:review-requested",
				},
				Messages: config.PhaseMessageConfig{
					Plan:      "osoba: 計画を作成します",
					Implement: "osoba: 実装を開始します",
					Review:    "osoba: レビューを開始します",
				},
			},
			Tmux: config.TmuxConfig{
				SessionPrefix: "osoba-",
			},
			Claude: &claude.ClaudeConfig{
				Phases: map[string]*claude.PhaseConfig{
					"plan": {
						Args:   []string{},
						Prompt: "Plan for issue {{issue-number}}: {{issue-title}}",
					},
					"implement": {
						Args:   []string{"--dangerously-skip-permissions"},
						Prompt: "Implement issue {{issue-number}}: {{issue-title}}",
					},
					"review": {
						Args:   []string{},
						Prompt: "Review implementation for issue {{issue-number}}",
					},
				},
			},
			Log: config.LogConfig{
				Level:  "info",
				Format: "text",
			},
		},
	}
}

// WithGitHubToken is deprecated and does nothing (gh command is used instead)
func (b *ConfigBuilder) WithGitHubToken(token string) *ConfigBuilder {
	// No-op: gh command is used for authentication
	return b
}

// WithPollingInterval sets the polling interval
func (b *ConfigBuilder) WithPollingInterval(interval time.Duration) *ConfigBuilder {
	b.cfg.GitHub.PollInterval = interval
	return b
}

// WithTmuxSessionPrefix sets the tmux session prefix
func (b *ConfigBuilder) WithTmuxSessionPrefix(prefix string) *ConfigBuilder {
	b.cfg.Tmux.SessionPrefix = prefix
	return b
}

// WithPlanPrompt sets the plan phase prompt
func (b *ConfigBuilder) WithPlanPrompt(prompt string) *ConfigBuilder {
	if b.cfg.Claude.Phases["plan"] != nil {
		b.cfg.Claude.Phases["plan"].Prompt = prompt
	}
	return b
}

// WithPlanArgs sets the plan phase args
func (b *ConfigBuilder) WithPlanArgs(args []string) *ConfigBuilder {
	if b.cfg.Claude.Phases["plan"] != nil {
		b.cfg.Claude.Phases["plan"].Args = args
	}
	return b
}

// WithImplementPrompt sets the implement phase prompt
func (b *ConfigBuilder) WithImplementPrompt(prompt string) *ConfigBuilder {
	if b.cfg.Claude.Phases["implement"] != nil {
		b.cfg.Claude.Phases["implement"].Prompt = prompt
	}
	return b
}

// WithImplementArgs sets the implement phase args
func (b *ConfigBuilder) WithImplementArgs(args []string) *ConfigBuilder {
	if b.cfg.Claude.Phases["implement"] != nil {
		b.cfg.Claude.Phases["implement"].Args = args
	}
	return b
}

// WithReviewPrompt sets the review phase prompt
func (b *ConfigBuilder) WithReviewPrompt(prompt string) *ConfigBuilder {
	if b.cfg.Claude.Phases["review"] != nil {
		b.cfg.Claude.Phases["review"].Prompt = prompt
	}
	return b
}

// WithReviewArgs sets the review phase args
func (b *ConfigBuilder) WithReviewArgs(args []string) *ConfigBuilder {
	if b.cfg.Claude.Phases["review"] != nil {
		b.cfg.Claude.Phases["review"].Args = args
	}
	return b
}

// Build returns the constructed Config
func (b *ConfigBuilder) Build() *config.Config {
	// Return a copy to prevent external modification
	cfgCopy := *b.cfg

	// Deep copy Claude config
	if b.cfg.Claude != nil {
		cfgCopy.Claude = &claude.ClaudeConfig{
			Phases: make(map[string]*claude.PhaseConfig),
		}
		for phase, config := range b.cfg.Claude.Phases {
			if config != nil {
				phaseCopy := *config
				if config.Args != nil {
					phaseCopy.Args = make([]string, len(config.Args))
					copy(phaseCopy.Args, config.Args)
				}
				cfgCopy.Claude.Phases[phase] = &phaseCopy
			}
		}
	}

	return &cfgCopy
}

// TemplateVariablesBuilder builds claude.TemplateVariables instances for testing
type TemplateVariablesBuilder struct {
	vars *claude.TemplateVariables
}

// NewTemplateVariablesBuilder creates a new TemplateVariablesBuilder with sensible defaults
func NewTemplateVariablesBuilder() *TemplateVariablesBuilder {
	return &TemplateVariablesBuilder{
		vars: &claude.TemplateVariables{
			IssueNumber: 1,
			IssueTitle:  "Default Issue",
			RepoName:    "test-repo",
		},
	}
}

// WithIssueNumber sets the issue number
func (b *TemplateVariablesBuilder) WithIssueNumber(number int) *TemplateVariablesBuilder {
	b.vars.IssueNumber = number
	return b
}

// WithIssueTitle sets the issue title
func (b *TemplateVariablesBuilder) WithIssueTitle(title string) *TemplateVariablesBuilder {
	b.vars.IssueTitle = title
	return b
}

// WithRepoName sets the repository name
func (b *TemplateVariablesBuilder) WithRepoName(name string) *TemplateVariablesBuilder {
	b.vars.RepoName = name
	return b
}

// FromIssue populates variables from a GitHub issue
func (b *TemplateVariablesBuilder) FromIssue(issue *github.Issue) *TemplateVariablesBuilder {
	if issue.Number != nil {
		b.vars.IssueNumber = *issue.Number
	}
	if issue.Title != nil {
		b.vars.IssueTitle = *issue.Title
	}
	return b
}

// Build returns the constructed TemplateVariables
func (b *TemplateVariablesBuilder) Build() *claude.TemplateVariables {
	// Return a copy to prevent external modification
	varsCopy := *b.vars
	return &varsCopy
}
