package builders_test

import (
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/testutil/builders"
	"github.com/stretchr/testify/assert"
)

func TestConfigBuilder(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		config := builders.NewConfigBuilder().Build()

		assert.NotNil(t, config)
		assert.Equal(t, 5*time.Minute, config.GitHub.PollInterval)
		assert.Equal(t, "osoba-", config.Tmux.SessionPrefix)
		assert.NotNil(t, config.Claude)
		assert.NotEmpty(t, config.Claude.Phases["plan"].Prompt)
		assert.NotEmpty(t, config.Claude.Phases["implement"].Prompt)
		assert.NotEmpty(t, config.Claude.Phases["review"].Prompt)
	})

	t.Run("custom github config", func(t *testing.T) {
		config := builders.NewConfigBuilder().
			WithGitHubToken("CUSTOM_TOKEN").
			WithPollingInterval(10 * time.Minute).
			Build()

		assert.Equal(t, 10*time.Minute, config.GitHub.PollInterval)
	})

	t.Run("custom tmux config", func(t *testing.T) {
		config := builders.NewConfigBuilder().
			WithTmuxSessionPrefix("myproject-").
			Build()

		assert.Equal(t, "myproject-", config.Tmux.SessionPrefix)
	})

	t.Run("custom phase configs", func(t *testing.T) {
		config := builders.NewConfigBuilder().
			WithPlanPrompt("Custom plan prompt {{issue-number}}").
			WithImplementPrompt("Custom implement prompt {{issue-title}}").
			WithReviewPrompt("Custom review prompt {{repo-name}}").
			Build()

		assert.Equal(t, "Custom plan prompt {{issue-number}}", config.Claude.Phases["plan"].Prompt)
		assert.Equal(t, "Custom implement prompt {{issue-title}}", config.Claude.Phases["implement"].Prompt)
		assert.Equal(t, "Custom review prompt {{repo-name}}", config.Claude.Phases["review"].Prompt)
	})

	t.Run("with claude args", func(t *testing.T) {
		config := builders.NewConfigBuilder().
			WithPlanArgs([]string{"--model", "claude-3-opus"}).
			WithImplementArgs([]string{"--dangerously-skip-permissions"}).
			Build()

		assert.Equal(t, []string{"--model", "claude-3-opus"}, config.Claude.Phases["plan"].Args)
		assert.Equal(t, []string{"--dangerously-skip-permissions"}, config.Claude.Phases["implement"].Args)
	})
}

func TestTemplateVariablesBuilder(t *testing.T) {
	t.Run("default variables", func(t *testing.T) {
		vars := builders.NewTemplateVariablesBuilder().Build()

		assert.NotNil(t, vars)
		assert.Equal(t, 1, vars.IssueNumber)
		assert.Equal(t, "Default Issue", vars.IssueTitle)
		assert.Equal(t, "test-repo", vars.RepoName)
	})

	t.Run("custom variables", func(t *testing.T) {
		vars := builders.NewTemplateVariablesBuilder().
			WithIssueNumber(123).
			WithIssueTitle("Custom Issue Title").
			WithRepoName("my-awesome-repo").
			Build()

		assert.Equal(t, 123, vars.IssueNumber)
		assert.Equal(t, "Custom Issue Title", vars.IssueTitle)
		assert.Equal(t, "my-awesome-repo", vars.RepoName)
	})

	t.Run("from issue", func(t *testing.T) {
		issue := builders.NewIssueBuilder().
			WithNumber(456).
			WithTitle("Issue from builder").
			Build()

		vars := builders.NewTemplateVariablesBuilder().
			FromIssue(issue).
			WithRepoName("repo-name").
			Build()

		assert.Equal(t, 456, vars.IssueNumber)
		assert.Equal(t, "Issue from builder", vars.IssueTitle)
		assert.Equal(t, "repo-name", vars.RepoName)
	})
}
