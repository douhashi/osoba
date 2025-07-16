// DISABLED: 古いgo-github APIベースのテストのため一時的に無効化
//go:build ignore
// +build ignore

package github

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-github/v67/github"
	"github.com/stretchr/testify/assert"
)

func TestLabelManagerWithRetry_TransitionLabelWithInfoWithRetry(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		issueNumber    int
		maxRetries     int
		expectedResult bool
		expectedInfo   *TransitionInfo
		expectedError  bool
	}{
		{
			name:           "success on first attempt",
			issueNumber:    123,
			maxRetries:     3,
			expectedResult: true,
			expectedInfo: &TransitionInfo{
				From: "status:needs-plan",
				To:   "status:planning",
			},
			expectedError: false,
		},
		{
			name:           "success on retry",
			issueNumber:    456,
			maxRetries:     3,
			expectedResult: true,
			expectedInfo: &TransitionInfo{
				From: "status:ready",
				To:   "status:implementing",
			},
			expectedError: false,
		},
		{
			name:           "no transition needed",
			issueNumber:    789,
			maxRetries:     3,
			expectedResult: false,
			expectedInfo:   nil,
			expectedError:  false,
		},
		{
			name:           "failure after max retries",
			issueNumber:    999,
			maxRetries:     2,
			expectedResult: false,
			expectedInfo:   nil,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Create a real LabelManager with a mock service
			mockService := new(MockLabelService)

			// Setup mock service based on what the mock label manager expects
			if tt.name == "success on first attempt" {
				mockService.On("ListLabelsByIssue", ctx, "owner", "repo", 123, (*github.ListOptions)(nil)).
					Return([]*github.Label{
						{Name: github.String("status:needs-plan")},
					}, &github.Response{}, nil)
				mockService.On("RemoveLabelForIssue", ctx, "owner", "repo", 123, "status:needs-plan").
					Return(&github.Response{}, nil)
				mockService.On("AddLabelsToIssue", ctx, "owner", "repo", 123, []string{"status:planning"}).
					Return([]*github.Label{}, &github.Response{}, nil)
			} else if tt.name == "success on retry" {
				// First attempt
				mockService.On("ListLabelsByIssue", ctx, "owner", "repo", 456, (*github.ListOptions)(nil)).
					Return([]*github.Label{}, &github.Response{}, errors.New("temporary error")).Once()
				// Second attempt
				mockService.On("ListLabelsByIssue", ctx, "owner", "repo", 456, (*github.ListOptions)(nil)).
					Return([]*github.Label{
						{Name: github.String("status:ready")},
					}, &github.Response{}, nil).Once()
				mockService.On("RemoveLabelForIssue", ctx, "owner", "repo", 456, "status:ready").
					Return(&github.Response{}, nil)
				mockService.On("AddLabelsToIssue", ctx, "owner", "repo", 456, []string{"status:implementing"}).
					Return([]*github.Label{}, &github.Response{}, nil)
			} else if tt.name == "no transition needed" {
				mockService.On("ListLabelsByIssue", ctx, "owner", "repo", 789, (*github.ListOptions)(nil)).
					Return([]*github.Label{
						{Name: github.String("bug")},
					}, &github.Response{}, nil)
			} else if tt.name == "failure after max retries" {
				mockService.On("ListLabelsByIssue", ctx, "owner", "repo", 999, (*github.ListOptions)(nil)).
					Return([]*github.Label{}, &github.Response{}, errors.New("persistent error"))
			}

			realLabelManager := NewGHLabelManager(mockService)
			lm := &LabelManagerWithRetry{
				LabelManager: realLabelManager,
				maxRetries:   tt.maxRetries,
				retryDelay:   10 * time.Millisecond, // Short delay for tests
			}

			result, info, err := lm.TransitionLabelWithInfoWithRetry(ctx, "owner", "repo", tt.issueNumber)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedResult, result)

			if tt.expectedInfo != nil {
				assert.NotNil(t, info)
				assert.Equal(t, tt.expectedInfo.From, info.From)
				assert.Equal(t, tt.expectedInfo.To, info.To)
			} else {
				assert.Nil(t, info)
			}

			mockService.AssertExpectations(t)
		})
	}
}
