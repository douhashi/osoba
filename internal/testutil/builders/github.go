package builders

import (
	"time"

	"github.com/douhashi/osoba/internal/github"
)

// IssueBuilder builds github.Issue instances for testing
type IssueBuilder struct {
	issue *github.Issue
}

// NewIssueBuilder creates a new IssueBuilder with sensible defaults
func NewIssueBuilder() *IssueBuilder {
	now := time.Now()
	return &IssueBuilder{
		issue: &github.Issue{
			Number:    github.Int(1),
			State:     github.String("open"),
			Title:     github.String("Default Issue"),
			Body:      github.String(""),
			Labels:    []*github.Label{},
			CreatedAt: &now,
			UpdatedAt: &now,
		},
	}
}

// WithNumber sets the issue number
func (b *IssueBuilder) WithNumber(number int) *IssueBuilder {
	b.issue.Number = github.Int(number)
	return b
}

// WithState sets the issue state
func (b *IssueBuilder) WithState(state string) *IssueBuilder {
	b.issue.State = github.String(state)
	return b
}

// WithTitle sets the issue title
func (b *IssueBuilder) WithTitle(title string) *IssueBuilder {
	b.issue.Title = github.String(title)
	return b
}

// WithBody sets the issue body
func (b *IssueBuilder) WithBody(body string) *IssueBuilder {
	b.issue.Body = github.String(body)
	return b
}

// WithLabels sets the issue labels
func (b *IssueBuilder) WithLabels(labels []string) *IssueBuilder {
	b.issue.Labels = make([]*github.Label, len(labels))
	for i, label := range labels {
		b.issue.Labels[i] = &github.Label{
			Name: github.String(label),
		}
	}
	return b
}

// WithLabel adds a single label to the issue
func (b *IssueBuilder) WithLabel(label string) *IssueBuilder {
	b.issue.Labels = append(b.issue.Labels, &github.Label{
		Name: github.String(label),
	})
	return b
}

// WithStatusLabel adds a status label to the issue
func (b *IssueBuilder) WithStatusLabel(status string) *IssueBuilder {
	return b.WithLabel("status:" + status)
}

// WithPriorityLabel adds a priority label to the issue
func (b *IssueBuilder) WithPriorityLabel(priority string) *IssueBuilder {
	return b.WithLabel("priority:" + priority)
}

// WithUser sets the issue user
func (b *IssueBuilder) WithUser(login string) *IssueBuilder {
	b.issue.User = &github.User{
		Login: github.String(login),
	}
	return b
}

// WithCreatedAt sets the creation time
func (b *IssueBuilder) WithCreatedAt(t time.Time) *IssueBuilder {
	b.issue.CreatedAt = &t
	return b
}

// WithUpdatedAt sets the update time
func (b *IssueBuilder) WithUpdatedAt(t time.Time) *IssueBuilder {
	b.issue.UpdatedAt = &t
	return b
}

// WithHTMLURL sets the HTML URL
func (b *IssueBuilder) WithHTMLURL(url string) *IssueBuilder {
	b.issue.HTMLURL = github.String(url)
	return b
}

// Build returns the constructed Issue
func (b *IssueBuilder) Build() *github.Issue {
	// Return a copy to prevent external modification
	issueCopy := *b.issue
	if b.issue.Labels != nil {
		issueCopy.Labels = make([]*github.Label, len(b.issue.Labels))
		for i, label := range b.issue.Labels {
			if label != nil {
				labelCopy := *label
				issueCopy.Labels[i] = &labelCopy
			}
		}
	}
	if b.issue.User != nil {
		userCopy := *b.issue.User
		issueCopy.User = &userCopy
	}
	return &issueCopy
}

// RepositoryBuilder builds github.Repository instances for testing
type RepositoryBuilder struct {
	repo *github.Repository
}

// NewRepositoryBuilder creates a new RepositoryBuilder with sensible defaults
func NewRepositoryBuilder() *RepositoryBuilder {
	return &RepositoryBuilder{
		repo: &github.Repository{
			Name: github.String("test-repo"),
			Owner: &github.User{
				Login: github.String("test-owner"),
			},
			Private:     github.Bool(false),
			Description: github.String(""),
		},
	}
}

// WithName sets the repository name
func (b *RepositoryBuilder) WithName(name string) *RepositoryBuilder {
	b.repo.Name = github.String(name)
	return b
}

// WithOwner sets the repository owner
func (b *RepositoryBuilder) WithOwner(owner string) *RepositoryBuilder {
	b.repo.Owner = &github.User{
		Login: github.String(owner),
	}
	return b
}

// WithDescription sets the repository description
func (b *RepositoryBuilder) WithDescription(desc string) *RepositoryBuilder {
	b.repo.Description = github.String(desc)
	return b
}

// WithPrivate sets whether the repository is private
func (b *RepositoryBuilder) WithPrivate(private bool) *RepositoryBuilder {
	b.repo.Private = github.Bool(private)
	return b
}

// WithDefaultBranch sets the default branch
func (b *RepositoryBuilder) WithDefaultBranch(branch string) *RepositoryBuilder {
	// DefaultBranch is not available in the current Repository struct
	// This method is kept for API compatibility but does nothing
	return b
}

// WithCloneURL sets the clone URL
func (b *RepositoryBuilder) WithCloneURL(url string) *RepositoryBuilder {
	// CloneURL is not available in the current Repository struct
	// This method is kept for API compatibility but does nothing
	return b
}

// WithHTMLURL sets the HTML URL
func (b *RepositoryBuilder) WithHTMLURL(url string) *RepositoryBuilder {
	b.repo.HTMLURL = github.String(url)
	return b
}

// WithFullName sets the repository full name (owner/repo)
func (b *RepositoryBuilder) WithFullName(fullName string) *RepositoryBuilder {
	// FullName is not available in the current Repository struct
	// This method is kept for API compatibility but does nothing
	return b
}

// AsArchived sets the repository as archived
func (b *RepositoryBuilder) AsArchived() *RepositoryBuilder {
	// Archived is not available in the current Repository struct
	// This method is kept for API compatibility but does nothing
	return b
}

// Build returns the constructed Repository
func (b *RepositoryBuilder) Build() *github.Repository {
	// Return a copy to prevent external modification
	repoCopy := *b.repo
	if b.repo.Owner != nil {
		ownerCopy := *b.repo.Owner
		repoCopy.Owner = &ownerCopy
	}
	return &repoCopy
}

// LabelBuilder builds github.Label instances for testing
type LabelBuilder struct {
	label *github.Label
}

// NewLabelBuilder creates a new LabelBuilder with sensible defaults
func NewLabelBuilder() *LabelBuilder {
	return &LabelBuilder{
		label: &github.Label{
			Name:        github.String("label"),
			Color:       github.String("0366d6"), // GitHub blue
			Description: github.String(""),
		},
	}
}

// WithName sets the label name
func (b *LabelBuilder) WithName(name string) *LabelBuilder {
	b.label.Name = github.String(name)
	return b
}

// WithColor sets the label color
func (b *LabelBuilder) WithColor(color string) *LabelBuilder {
	b.label.Color = github.String(color)
	return b
}

// WithDescription sets the label description
func (b *LabelBuilder) WithDescription(desc string) *LabelBuilder {
	b.label.Description = github.String(desc)
	return b
}

// AsStatusLabel creates a status label with appropriate color
func (b *LabelBuilder) AsStatusLabel(status string) *github.Label {
	b.label.Name = github.String("status:" + status)
	switch status {
	case "ready", "implementing", "completed":
		b.label.Color = github.String("0e8a16") // Green
	case "needs-plan", "planning":
		b.label.Color = github.String("fbca04") // Yellow
	case "review-requested", "reviewing":
		b.label.Color = github.String("1d76db") // Blue
	default:
		b.label.Color = github.String("d876e3") // Purple
	}
	return b.Build()
}

// AsPriorityLabel creates a priority label with appropriate color
func (b *LabelBuilder) AsPriorityLabel(priority string) *github.Label {
	b.label.Name = github.String("priority:" + priority)
	switch priority {
	case "high":
		b.label.Color = github.String("b60205") // Red
	case "medium":
		b.label.Color = github.String("fbca04") // Yellow
	case "low":
		b.label.Color = github.String("0e8a16") // Green
	default:
		b.label.Color = github.String("d876e3") // Purple
	}
	return b.Build()
}

// Build returns the constructed Label
func (b *LabelBuilder) Build() *github.Label {
	// Return a copy to prevent external modification
	labelCopy := *b.label
	return &labelCopy
}

// RateLimitsBuilder builds github.RateLimits instances for testing
type RateLimitsBuilder struct {
	rateLimits *github.RateLimits
}

// NewRateLimitsBuilder creates a new RateLimitsBuilder with sensible defaults
func NewRateLimitsBuilder() *RateLimitsBuilder {
	reset := time.Now().Add(1 * time.Hour)
	return &RateLimitsBuilder{
		rateLimits: &github.RateLimits{
			Core: &github.RateLimit{
				Limit:     5000,
				Remaining: 4999,
				Reset:     reset,
			},
			Search: &github.RateLimit{
				Limit:     30,
				Remaining: 30,
				Reset:     reset,
			},
		},
	}
}

// WithCoreLimit sets the core API rate limit
func (b *RateLimitsBuilder) WithCoreLimit(limit, remaining int) *RateLimitsBuilder {
	b.rateLimits.Core.Limit = limit
	b.rateLimits.Core.Remaining = remaining
	return b
}

// WithSearchLimit sets the search API rate limit
func (b *RateLimitsBuilder) WithSearchLimit(limit, remaining int) *RateLimitsBuilder {
	b.rateLimits.Search.Limit = limit
	b.rateLimits.Search.Remaining = remaining
	return b
}

// AsExhausted sets all rate limits to exhausted
func (b *RateLimitsBuilder) AsExhausted() *RateLimitsBuilder {
	b.rateLimits.Core.Remaining = 0
	b.rateLimits.Search.Remaining = 0
	b.rateLimits.Core.Reset = time.Now().Add(30 * time.Minute)
	b.rateLimits.Search.Reset = time.Now().Add(30 * time.Minute)
	return b
}

// Build returns the constructed RateLimits
func (b *RateLimitsBuilder) Build() *github.RateLimits {
	// Return a copy to prevent external modification
	rateLimitsCopy := *b.rateLimits
	if b.rateLimits.Core != nil {
		coreCopy := *b.rateLimits.Core
		rateLimitsCopy.Core = &coreCopy
	}
	if b.rateLimits.Search != nil {
		searchCopy := *b.rateLimits.Search
		rateLimitsCopy.Search = &searchCopy
	}
	return &rateLimitsCopy
}
