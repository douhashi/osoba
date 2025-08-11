package watcher

import (
	"context"
	"testing"

	gh "github.com/douhashi/osoba/internal/github"
	"github.com/stretchr/testify/assert"
)

func TestNoOpAction_Execute(t *testing.T) {
	tests := []struct {
		name        string
		issue       *gh.Issue
		expectError bool
	}{
		{
			name: "successful no-op execution",
			issue: &gh.Issue{
				Number: intPtr(222),
				Title:  stringPtr("Test Issue"),
				Labels: []*gh.Label{
					{Name: stringPtr("status:requires-changes")},
				},
			},
			expectError: false,
		},
		{
			name:        "nil issue",
			issue:       nil,
			expectError: false, // NoOp should handle nil gracefully
		},
		{
			name: "issue without number",
			issue: &gh.Issue{
				Title: stringPtr("Test Issue"),
			},
			expectError: false, // NoOp should handle missing fields gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := NewNoOpAction(NewMockLogger())
			err := action.Execute(context.Background(), tt.issue)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNoOpAction_CanExecute(t *testing.T) {
	tests := []struct {
		name           string
		issue          *gh.Issue
		expectedResult bool
	}{
		{
			name: "can execute with valid issue",
			issue: &gh.Issue{
				Number: intPtr(222),
			},
			expectedResult: true,
		},
		{
			name:           "cannot execute with nil issue",
			issue:          nil,
			expectedResult: false,
		},
		{
			name: "cannot execute without issue number",
			issue: &gh.Issue{
				Title: stringPtr("Test Issue"),
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := NewNoOpAction(NewMockLogger())
			result := action.CanExecute(tt.issue)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestNoOpAction_String(t *testing.T) {
	action := NewNoOpAction(NewMockLogger())
	assert.Equal(t, "NoOpAction", action.String())
}
