package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLabelManagerWithRetry is a mock for testing
type MockLabelManagerWithRetry struct {
	mock.Mock
}

func (m *MockLabelManagerWithRetry) TransitionLabelWithRetry(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockLabelManagerWithRetry) TransitionLabelWithInfoWithRetry(ctx context.Context, owner, repo string, issueNumber int) (bool, *TransitionInfo, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	if args.Get(1) == nil {
		return args.Bool(0), nil, args.Error(2)
	}
	return args.Bool(0), args.Get(1).(*TransitionInfo), args.Error(2)
}

func (m *MockLabelManagerWithRetry) EnsureLabelsExistWithRetry(ctx context.Context, owner, repo string) error {
	args := m.Called(ctx, owner, repo)
	return args.Error(0)
}

func TestClient_TransitionIssueLabelWithInfo(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		owner          string
		repo           string
		issueNumber    int
		expectedResult bool
		expectedInfo   *TransitionInfo
		expectedError  bool
		setupMock      func(*MockLabelManagerWithRetry)
	}{
		{
			name:           "successful transition",
			owner:          "test-owner",
			repo:           "test-repo",
			issueNumber:    123,
			expectedResult: true,
			expectedInfo: &TransitionInfo{
				FromLabel: "status:needs-plan",
				ToLabel:   "status:planning",
			},
			expectedError: false,
			setupMock: func(m *MockLabelManagerWithRetry) {
				m.On("TransitionLabelWithInfoWithRetry", ctx, "test-owner", "test-repo", 123).
					Return(true, &TransitionInfo{
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
			setupMock: func(m *MockLabelManagerWithRetry) {
				m.On("TransitionLabelWithInfoWithRetry", ctx, "test-owner", "test-repo", 456).
					Return(false, (*TransitionInfo)(nil), nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock
			mockLabelManager := new(MockLabelManagerWithRetry)
			tt.setupMock(mockLabelManager)

			// Create client with mock
			client := &Client{
				labelManager: mockLabelManager,
			}

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
