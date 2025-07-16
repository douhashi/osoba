package github

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Issue represents a GitHub issue.
type Issue struct {
	ID        *int64     `json:"id,omitempty"`
	Number    *int       `json:"number,omitempty"`
	Title     *string    `json:"title,omitempty"`
	Body      *string    `json:"body,omitempty"`
	State     *string    `json:"state,omitempty"`
	User      *User      `json:"user,omitempty"`
	Labels    []*Label   `json:"labels,omitempty"`
	Assignee  *User      `json:"assignee,omitempty"`
	Assignees []*User    `json:"assignees,omitempty"`
	Milestone *Milestone `json:"milestone,omitempty"`
	Comments  *int       `json:"comments,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
	HTMLURL   *string    `json:"html_url,omitempty"`
}

// Label represents a GitHub label on an Issue.
type Label struct {
	ID          *int64  `json:"id,omitempty"`
	URL         *string `json:"url,omitempty"`
	Name        *string `json:"name,omitempty"`
	Color       *string `json:"color,omitempty"`
	Description *string `json:"description,omitempty"`
}

// User represents a GitHub user.
type User struct {
	Login     *string `json:"login,omitempty"`
	ID        *int64  `json:"id,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	HTMLURL   *string `json:"html_url,omitempty"`
	Type      *string `json:"type,omitempty"`
}

// Repository represents a GitHub repository.
type Repository struct {
	ID          *int64  `json:"id,omitempty"`
	Name        *string `json:"name,omitempty"`
	FullName    *string `json:"full_name,omitempty"`
	Owner       *User   `json:"owner,omitempty"`
	Private     *bool   `json:"private,omitempty"`
	Description *string `json:"description,omitempty"`
	Fork        *bool   `json:"fork,omitempty"`
	HTMLURL     *string `json:"html_url,omitempty"`
}

// Milestone represents a GitHub milestone.
type Milestone struct {
	ID          *int64     `json:"id,omitempty"`
	Number      *int       `json:"number,omitempty"`
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	State       *string    `json:"state,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	DueOn       *time.Time `json:"due_on,omitempty"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
}

// IssueComment represents a comment on a GitHub issue.
type IssueComment struct {
	ID        *int64     `json:"id,omitempty"`
	Body      *string    `json:"body,omitempty"`
	User      *User      `json:"user,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	HTMLURL   *string    `json:"html_url,omitempty"`
}

// RateLimits represents the rate limits for different GitHub API categories.
type RateLimits struct {
	Core     *RateLimit `json:"core,omitempty"`
	Search   *RateLimit `json:"search,omitempty"`
	GraphQL  *RateLimit `json:"graphql,omitempty"`
	Actions  *RateLimit `json:"actions,omitempty"`
	Packages *RateLimit `json:"packages,omitempty"`
}

// RateLimit represents the rate limit for a specific GitHub API category.
type RateLimit struct {
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	Reset     time.Time `json:"reset"`
}

// Response represents a GitHub API response.
type Response struct {
	*http.Response
	NextPage  int
	PrevPage  int
	FirstPage int
	LastPage  int
}

// ListOptions specifies optional parameters to methods that support pagination.
type ListOptions struct {
	Page    int `url:"page,omitempty"`
	PerPage int `url:"per_page,omitempty"`
}

// IssueListByRepoOptions specifies optional parameters to the IssuesService.ListByRepo method.
type IssueListByRepoOptions struct {
	State     string    `url:"state,omitempty"`
	Labels    []string  `url:"labels,comma,omitempty"`
	Sort      string    `url:"sort,omitempty"`
	Direction string    `url:"direction,omitempty"`
	Since     time.Time `url:"since,omitempty"`
	ListOptions
}

// ErrorResponse represents an error response from the GitHub API.
type ErrorResponse struct {
	Message string  `json:"message"`
	Errors  []Error `json:"errors,omitempty"`
}

// Error returns the error message.
func (r *ErrorResponse) Error() string {
	return r.Message
}

// Error represents an individual error in an ErrorResponse.
type Error struct {
	Resource string `json:"resource,omitempty"`
	Field    string `json:"field,omitempty"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message,omitempty"`
}

// RateLimitError occurs when GitHub API rate limit is exceeded.
type RateLimitError struct {
	Rate     RateLimit
	Response *http.Response
	Message  string
}

// Error returns the error message.
func (r *RateLimitError) Error() string {
	return fmt.Sprintf("GitHub API rate limit exceeded. Reset at %v", r.Rate.Reset)
}

// Helper functions for creating pointers to basic types

// String returns a pointer to the given string value.
func String(v string) *string {
	return &v
}

// Int returns a pointer to the given int value.
func Int(v int) *int {
	return &v
}

// Int64 returns a pointer to the given int64 value.
func Int64(v int64) *int64 {
	return &v
}

// Bool returns a pointer to the given bool value.
func Bool(v bool) *bool {
	return &v
}

// LabelDefinition defines a GitHub label with its properties
type LabelDefinition struct {
	Name        string
	Color       string
	Description string
}

// LabelManagerInterface defines the interface for label management operations
type LabelManagerInterface interface {
	TransitionLabelWithRetry(ctx context.Context, owner, repo string, issueNumber int) (bool, error)
	TransitionLabelWithInfoWithRetry(ctx context.Context, owner, repo string, issueNumber int) (bool, *TransitionInfo, error)
	EnsureLabelsExistWithRetry(ctx context.Context, owner, repo string) error
}

// LabelService defines the interface for GitHub label operations (deprecated)
type LabelService interface{}

// LabelManager is deprecated, use GHLabelManager instead
type LabelManager struct{}

// TransitionInfo represents the result of a label transition operation
type TransitionInfo struct {
	TransitionFound bool
	FromLabel       string
	ToLabel         string
	CurrentLabels   []string
}
