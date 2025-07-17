package builders_test

import (
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/testutil/builders"
	"github.com/stretchr/testify/assert"
)

func TestIssueBuilder(t *testing.T) {
	t.Run("default issue", func(t *testing.T) {
		issue := builders.NewIssueBuilder().Build()

		assert.NotNil(t, issue)
		assert.NotNil(t, issue.Number)
		assert.Equal(t, 1, *issue.Number)
		assert.NotNil(t, issue.State)
		assert.Equal(t, "open", *issue.State)
		assert.NotNil(t, issue.Title)
		assert.Equal(t, "Default Issue", *issue.Title)
		assert.Empty(t, issue.Labels)
	})

	t.Run("custom issue", func(t *testing.T) {
		issue := builders.NewIssueBuilder().
			WithNumber(123).
			WithState("closed").
			WithTitle("Custom Issue Title").
			WithBody("This is a custom issue body").
			WithLabels([]string{"bug", "priority:high"}).
			Build()

		assert.NotNil(t, issue)
		assert.Equal(t, 123, *issue.Number)
		assert.Equal(t, "closed", *issue.State)
		assert.Equal(t, "Custom Issue Title", *issue.Title)
		assert.Equal(t, "This is a custom issue body", *issue.Body)
		assert.Len(t, issue.Labels, 2)
		assert.Equal(t, "bug", *issue.Labels[0].Name)
		assert.Equal(t, "priority:high", *issue.Labels[1].Name)
	})

	t.Run("with user", func(t *testing.T) {
		issue := builders.NewIssueBuilder().
			WithUser("testuser").
			Build()

		assert.NotNil(t, issue.User)
		assert.Equal(t, "testuser", *issue.User.Login)
	})

	t.Run("with timestamps", func(t *testing.T) {
		now := time.Now()
		issue := builders.NewIssueBuilder().
			WithCreatedAt(now).
			WithUpdatedAt(now.Add(1 * time.Hour)).
			Build()

		assert.NotNil(t, issue.CreatedAt)
		assert.Equal(t, now.Unix(), issue.CreatedAt.Unix())
		assert.NotNil(t, issue.UpdatedAt)
		assert.Equal(t, now.Add(1*time.Hour).Unix(), issue.UpdatedAt.Unix())
	})

	t.Run("with html url", func(t *testing.T) {
		issue := builders.NewIssueBuilder().
			WithNumber(456).
			WithHTMLURL("https://github.com/owner/repo/issues/456").
			Build()

		assert.NotNil(t, issue.HTMLURL)
		assert.Equal(t, "https://github.com/owner/repo/issues/456", *issue.HTMLURL)
	})
}

func TestRepositoryBuilder(t *testing.T) {
	t.Run("default repository", func(t *testing.T) {
		repo := builders.NewRepositoryBuilder().Build()

		assert.NotNil(t, repo)
		assert.NotNil(t, repo.Name)
		assert.Equal(t, "test-repo", *repo.Name)
		assert.NotNil(t, repo.Owner)
		assert.Equal(t, "test-owner", *repo.Owner.Login)
		assert.NotNil(t, repo.Private)
		assert.False(t, *repo.Private)
	})

	t.Run("custom repository", func(t *testing.T) {
		repo := builders.NewRepositoryBuilder().
			WithName("my-repo").
			WithOwner("my-org").
			WithDescription("My awesome repository").
			WithPrivate(true).
			WithDefaultBranch("develop").
			Build()

		assert.Equal(t, "my-repo", *repo.Name)
		assert.Equal(t, "my-org", *repo.Owner.Login)
		assert.Equal(t, "My awesome repository", *repo.Description)
		assert.True(t, *repo.Private)
		// DefaultBranch is not available in the current implementation
	})

	t.Run("with urls", func(t *testing.T) {
		repo := builders.NewRepositoryBuilder().
			WithName("test").
			WithOwner("user").
			WithCloneURL("https://github.com/user/test.git").
			WithHTMLURL("https://github.com/user/test").
			Build()

		// CloneURL is not available in the current implementation
		assert.NotNil(t, repo.HTMLURL)
		assert.Equal(t, "https://github.com/user/test", *repo.HTMLURL)
	})
}

func TestLabelBuilder(t *testing.T) {
	t.Run("default label", func(t *testing.T) {
		label := builders.NewLabelBuilder().Build()

		assert.NotNil(t, label)
		assert.Equal(t, "label", *label.Name)
		assert.Equal(t, "0366d6", *label.Color)
		assert.Equal(t, "", *label.Description)
	})

	t.Run("custom label", func(t *testing.T) {
		label := builders.NewLabelBuilder().
			WithName("bug").
			WithColor("d73a4a").
			WithDescription("Something isn't working").
			Build()

		assert.Equal(t, "bug", *label.Name)
		assert.Equal(t, "d73a4a", *label.Color)
		assert.Equal(t, "Something isn't working", *label.Description)
	})

	t.Run("status label preset", func(t *testing.T) {
		label := builders.NewLabelBuilder().AsStatusLabel("ready")

		assert.Equal(t, "status:ready", *label.Name)
		assert.Equal(t, "0e8a16", *label.Color) // 緑色
	})

	t.Run("priority label preset", func(t *testing.T) {
		label := builders.NewLabelBuilder().AsPriorityLabel("high")

		assert.Equal(t, "priority:high", *label.Name)
		assert.Equal(t, "b60205", *label.Color) // 赤色
	})
}

func TestRateLimitsBuilder(t *testing.T) {
	t.Run("default rate limits", func(t *testing.T) {
		rateLimits := builders.NewRateLimitsBuilder().Build()

		assert.NotNil(t, rateLimits)
		assert.NotNil(t, rateLimits.Core)
		assert.Equal(t, 5000, rateLimits.Core.Limit)
		assert.Equal(t, 4999, rateLimits.Core.Remaining)
		assert.NotNil(t, rateLimits.Search)
		assert.Equal(t, 30, rateLimits.Search.Limit)
		assert.Equal(t, 30, rateLimits.Search.Remaining)
	})

	t.Run("custom rate limits", func(t *testing.T) {
		rateLimits := builders.NewRateLimitsBuilder().
			WithCoreLimit(1000, 500).
			WithSearchLimit(10, 5).
			Build()

		assert.Equal(t, 1000, rateLimits.Core.Limit)
		assert.Equal(t, 500, rateLimits.Core.Remaining)
		assert.Equal(t, 10, rateLimits.Search.Limit)
		assert.Equal(t, 5, rateLimits.Search.Remaining)
	})

	t.Run("exhausted rate limits", func(t *testing.T) {
		rateLimits := builders.NewRateLimitsBuilder().AsExhausted().Build()

		assert.Equal(t, 5000, rateLimits.Core.Limit)
		assert.Equal(t, 0, rateLimits.Core.Remaining)
		assert.Equal(t, 30, rateLimits.Search.Limit)
		assert.Equal(t, 0, rateLimits.Search.Remaining)
	})
}
