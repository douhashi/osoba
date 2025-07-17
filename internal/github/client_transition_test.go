package github_test

import (
	"context"
	"testing"

	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
)

func TestClient_TransitionIssueLabelWithInfo(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		owner          string
		repo           string
		issueNumber    int
		expectedResult bool
		expectedInfo   *github.TransitionInfo
		expectedError  bool
		setupMock      func(*mocks.MockLabelManagerInterface)
	}{
		{
			name:           "successful transition",
			owner:          "test-owner",
			repo:           "test-repo",
			issueNumber:    123,
			expectedResult: true,
			expectedInfo: &github.TransitionInfo{
				FromLabel: "status:needs-plan",
				ToLabel:   "status:planning",
			},
			expectedError: false,
			setupMock: func(m *mocks.MockLabelManagerInterface) {
				m.On("TransitionLabelWithInfoWithRetry", ctx, "test-owner", "test-repo", 123).
					Return(true, &github.TransitionInfo{
						FromLabel: "status:needs-plan",
						ToLabel:   "status:planning",
					}, nil)
			},
		},
		{
			name:           "no transition needed",
			owner:          "test-owner",
			repo:           "test-repo",
			issueNumber:    456,
			expectedResult: false,
			expectedInfo:   nil,
			expectedError:  false,
			setupMock: func(m *mocks.MockLabelManagerInterface) {
				m.On("TransitionLabelWithInfoWithRetry", ctx, "test-owner", "test-repo", 456).
					Return(false, (*github.TransitionInfo)(nil), nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock
			mockLabelManager := new(mocks.MockLabelManagerInterface)
			tt.setupMock(mockLabelManager)

			// Create client with mock
			client := github.NewClientWithLabelManager(mockLabelManager)

			// Execute
			result, info, err := client.TransitionIssueLabelWithInfo(ctx, tt.owner, tt.repo, tt.issueNumber)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedResult, result)

			if tt.expectedInfo != nil {
				assert.NotNil(t, info)
				assert.Equal(t, tt.expectedInfo.FromLabel, info.FromLabel)
				assert.Equal(t, tt.expectedInfo.ToLabel, info.ToLabel)
			} else {
				assert.Nil(t, info)
			}

			mockLabelManager.AssertExpectations(t)
		})
	}
}
