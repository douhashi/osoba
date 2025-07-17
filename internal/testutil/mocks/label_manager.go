package mocks

import (
	"context"

	"github.com/douhashi/osoba/internal/github"
	"github.com/stretchr/testify/mock"
)

// MockLabelManagerInterface is a mock implementation of github.LabelManagerInterface
type MockLabelManagerInterface struct {
	mock.Mock
}

// NewMockLabelManagerInterface creates a new instance of MockLabelManagerInterface
func NewMockLabelManagerInterface() *MockLabelManagerInterface {
	return &MockLabelManagerInterface{}
}

// WithDefaultBehavior sets up common default behaviors for the mock
func (m *MockLabelManagerInterface) WithDefaultBehavior() *MockLabelManagerInterface {
	// TransitionLabelWithRetry のデフォルト動作（何も遷移しない）
	m.On("TransitionLabelWithRetry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().Return(false, nil)

	// TransitionLabelWithInfoWithRetry のデフォルト動作
	m.On("TransitionLabelWithInfoWithRetry", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Maybe().Return(false, &github.TransitionInfo{
		TransitionFound: false,
		FromLabel:       "",
		ToLabel:         "",
		CurrentLabels:   []string{},
	}, nil)

	// EnsureLabelsExistWithRetry のデフォルト動作（成功）
	m.On("EnsureLabelsExistWithRetry", mock.Anything, mock.Anything, mock.Anything).
		Maybe().Return(nil)

	return m
}

// TransitionLabelWithRetry mocks the TransitionLabelWithRetry method
func (m *MockLabelManagerInterface) TransitionLabelWithRetry(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	return args.Bool(0), args.Error(1)
}

// TransitionLabelWithInfoWithRetry mocks the TransitionLabelWithInfoWithRetry method
func (m *MockLabelManagerInterface) TransitionLabelWithInfoWithRetry(ctx context.Context, owner, repo string, issueNumber int) (bool, *github.TransitionInfo, error) {
	args := m.Called(ctx, owner, repo, issueNumber)
	if args.Get(1) == nil {
		return args.Bool(0), nil, args.Error(2)
	}
	return args.Bool(0), args.Get(1).(*github.TransitionInfo), args.Error(2)
}

// EnsureLabelsExistWithRetry mocks the EnsureLabelsExistWithRetry method
func (m *MockLabelManagerInterface) EnsureLabelsExistWithRetry(ctx context.Context, owner, repo string) error {
	args := m.Called(ctx, owner, repo)
	return args.Error(0)
}

// WithSuccessfulTransition sets up mock to return a successful transition
func (m *MockLabelManagerInterface) WithSuccessfulTransition(owner, repo string, issueNumber int, fromLabel, toLabel string) *MockLabelManagerInterface {
	ctx := mock.Anything

	// TransitionLabelWithRetry を成功に設定
	m.On("TransitionLabelWithRetry", ctx, owner, repo, issueNumber).
		Return(true, nil)

	// TransitionLabelWithInfoWithRetry も成功に設定
	m.On("TransitionLabelWithInfoWithRetry", ctx, owner, repo, issueNumber).
		Return(true, &github.TransitionInfo{
			TransitionFound: true,
			FromLabel:       fromLabel,
			ToLabel:         toLabel,
			CurrentLabels:   []string{toLabel},
		}, nil)

	return m
}

// WithTransitionError sets up mock to return an error during transition
func (m *MockLabelManagerInterface) WithTransitionError(owner, repo string, issueNumber int, err error) *MockLabelManagerInterface {
	ctx := mock.Anything

	m.On("TransitionLabelWithRetry", ctx, owner, repo, issueNumber).
		Return(false, err)

	m.On("TransitionLabelWithInfoWithRetry", ctx, owner, repo, issueNumber).
		Return(false, nil, err)

	return m
}

// WithLabelsEnsured sets up mock for successful label ensuring
func (m *MockLabelManagerInterface) WithLabelsEnsured(owner, repo string) *MockLabelManagerInterface {
	m.On("EnsureLabelsExistWithRetry", mock.Anything, owner, repo).
		Return(nil)
	return m
}

// WithLabelsEnsureError sets up mock to return an error when ensuring labels
func (m *MockLabelManagerInterface) WithLabelsEnsureError(owner, repo string, err error) *MockLabelManagerInterface {
	m.On("EnsureLabelsExistWithRetry", mock.Anything, owner, repo).
		Return(err)
	return m
}

// Ensure MockLabelManagerInterface implements github.LabelManagerInterface
var _ github.LabelManagerInterface = (*MockLabelManagerInterface)(nil)
