package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v67/github"
)

// LabelDefinition defines a GitHub label with its properties
type LabelDefinition struct {
	Name        string
	Color       string
	Description string
}

// LabelService defines the interface for GitHub label operations
type LabelService interface {
	ListLabelsByIssue(ctx context.Context, owner, repo string, number int, opts *github.ListOptions) ([]*github.Label, *github.Response, error)
	AddLabelsToIssue(ctx context.Context, owner, repo string, number int, labels []string) ([]*github.Label, *github.Response, error)
	RemoveLabelForIssue(ctx context.Context, owner, repo string, number int, label string) (*github.Response, error)
	ListLabels(ctx context.Context, owner, repo string, opts *github.ListOptions) ([]*github.Label, *github.Response, error)
	CreateLabel(ctx context.Context, owner, repo string, label *github.Label) (*github.Label, *github.Response, error)
}

// LabelManager manages GitHub label operations and transitions
type LabelManager struct {
	client           LabelService
	labelDefinitions map[string]LabelDefinition
	transitionRules  map[string]string
}

// NewLabelManager creates a new LabelManager instance
func NewLabelManager(client LabelService) *LabelManager {
	lm := &LabelManager{
		client:           client,
		labelDefinitions: make(map[string]LabelDefinition),
		transitionRules:  make(map[string]string),
	}

	// Initialize label definitions
	lm.initializeLabelDefinitions()

	// Initialize transition rules
	lm.initializeTransitionRules()

	return lm
}

// initializeLabelDefinitions sets up the label definitions
func (lm *LabelManager) initializeLabelDefinitions() {
	// Trigger labels
	lm.labelDefinitions["status:needs-plan"] = LabelDefinition{
		Name:        "status:needs-plan",
		Color:       "0052cc",
		Description: "Planning phase required",
	}

	lm.labelDefinitions["status:ready"] = LabelDefinition{
		Name:        "status:ready",
		Color:       "0e8a16",
		Description: "Ready for implementation",
	}

	lm.labelDefinitions["status:review-requested"] = LabelDefinition{
		Name:        "status:review-requested",
		Color:       "d93f0b",
		Description: "Review requested",
	}

	// In-progress labels
	lm.labelDefinitions["status:planning"] = LabelDefinition{
		Name:        "status:planning",
		Color:       "0052cc",
		Description: "Currently in planning phase",
	}

	lm.labelDefinitions["status:implementing"] = LabelDefinition{
		Name:        "status:implementing",
		Color:       "0e8a16",
		Description: "Currently being implemented",
	}

	lm.labelDefinitions["status:reviewing"] = LabelDefinition{
		Name:        "status:reviewing",
		Color:       "d93f0b",
		Description: "Currently under review",
	}
}

// initializeTransitionRules sets up the label transition rules
func (lm *LabelManager) initializeTransitionRules() {
	lm.transitionRules["status:needs-plan"] = "status:planning"
	lm.transitionRules["status:ready"] = "status:implementing"
	lm.transitionRules["status:review-requested"] = "status:reviewing"
}

// GetLabelDefinitions returns all label definitions
func (lm *LabelManager) GetLabelDefinitions() map[string]LabelDefinition {
	return lm.labelDefinitions
}

// GetTransitionRules returns all transition rules
func (lm *LabelManager) GetTransitionRules() map[string]string {
	return lm.transitionRules
}

// TransitionLabel transitions an issue from a trigger label to an in-progress label
func (lm *LabelManager) TransitionLabel(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	// Get current labels
	labels, _, err := lm.client.ListLabelsByIssue(ctx, owner, repo, issueNumber, nil)
	if err != nil {
		return false, fmt.Errorf("failed to list labels: %w", err)
	}

	// Check if already has an in-progress label
	for _, label := range labels {
		labelName := *label.Name
		if lm.isInProgressLabel(labelName) {
			// Already in progress, skip transition
			return false, nil
		}
	}

	// Find trigger label and perform transition
	for _, label := range labels {
		labelName := *label.Name
		if targetLabel, exists := lm.transitionRules[labelName]; exists {
			// Remove trigger label
			_, err := lm.client.RemoveLabelForIssue(ctx, owner, repo, issueNumber, labelName)
			if err != nil {
				return false, fmt.Errorf("failed to remove label %s: %w", labelName, err)
			}

			// Add in-progress label
			_, _, err = lm.client.AddLabelsToIssue(ctx, owner, repo, issueNumber, []string{targetLabel})
			if err != nil {
				// Try to restore the original label
				lm.client.AddLabelsToIssue(ctx, owner, repo, issueNumber, []string{labelName})
				return false, fmt.Errorf("failed to add label %s: %w", targetLabel, err)
			}

			return true, nil
		}
	}

	// No trigger label found
	return false, nil
}

// isInProgressLabel checks if a label is an in-progress label
func (lm *LabelManager) isInProgressLabel(labelName string) bool {
	inProgressLabels := []string{
		"status:planning",
		"status:implementing",
		"status:reviewing",
	}

	for _, inProgressLabel := range inProgressLabels {
		if labelName == inProgressLabel {
			return true
		}
	}

	return false
}

// EnsureLabelsExist ensures all required labels exist in the repository
func (lm *LabelManager) EnsureLabelsExist(ctx context.Context, owner, repo string) error {
	// Get existing labels
	existingLabels, _, err := lm.client.ListLabels(ctx, owner, repo, nil)
	if err != nil {
		return fmt.Errorf("failed to list repository labels: %w", err)
	}

	// Create a map of existing labels for quick lookup
	existingLabelMap := make(map[string]bool)
	for _, label := range existingLabels {
		existingLabelMap[*label.Name] = true
	}

	// Create missing labels
	for _, labelDef := range lm.labelDefinitions {
		if !existingLabelMap[labelDef.Name] {
			// Label doesn't exist, create it
			newLabel := &github.Label{
				Name:        github.String(labelDef.Name),
				Color:       github.String(labelDef.Color),
				Description: github.String(labelDef.Description),
			}

			_, _, err := lm.client.CreateLabel(ctx, owner, repo, newLabel)
			if err != nil {
				return fmt.Errorf("failed to create label %s: %w", labelDef.Name, err)
			}
		}
	}

	return nil
}

// TransitionLabelWithInfo transitions an issue from a trigger label to an in-progress label and returns transition info
func (lm *LabelManager) TransitionLabelWithInfo(ctx context.Context, owner, repo string, issueNumber int) (bool, *TransitionInfo, error) {
	// Get current labels
	labels, _, err := lm.client.ListLabelsByIssue(ctx, owner, repo, issueNumber, nil)
	if err != nil {
		return false, nil, fmt.Errorf("failed to list labels: %w", err)
	}

	// Check if already has an in-progress label
	for _, label := range labels {
		labelName := *label.Name
		if lm.isInProgressLabel(labelName) {
			// Already in progress, skip transition
			return false, nil, nil
		}
	}

	// Find trigger label and perform transition
	for _, label := range labels {
		labelName := *label.Name
		if targetLabel, exists := lm.transitionRules[labelName]; exists {
			// Remove trigger label
			_, err := lm.client.RemoveLabelForIssue(ctx, owner, repo, issueNumber, labelName)
			if err != nil {
				return false, nil, fmt.Errorf("failed to remove label %s: %w", labelName, err)
			}

			// Add in-progress label
			_, _, err = lm.client.AddLabelsToIssue(ctx, owner, repo, issueNumber, []string{targetLabel})
			if err != nil {
				// Try to restore the original label
				lm.client.AddLabelsToIssue(ctx, owner, repo, issueNumber, []string{labelName})
				return false, nil, fmt.Errorf("failed to add label %s: %w", targetLabel, err)
			}

			// Return transition info
			return true, &TransitionInfo{
				From: labelName,
				To:   targetLabel,
			}, nil
		}
	}

	// No trigger label found
	return false, nil, nil
}
