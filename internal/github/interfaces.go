package github

import "context"

// LabelManagerInterface defines the interface for label management operations
type LabelManagerInterface interface {
	TransitionLabelWithRetry(ctx context.Context, owner, repo string, issueNumber int) (bool, error)
	TransitionLabelWithInfoWithRetry(ctx context.Context, owner, repo string, issueNumber int) (bool, *TransitionInfo, error)
	EnsureLabelsExistWithRetry(ctx context.Context, owner, repo string) error
}
