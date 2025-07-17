package mocks

import (
	"context"

	"github.com/douhashi/osoba/internal/git"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of git.Repository interface
type MockRepository struct {
	mock.Mock
}

// NewMockRepository creates a new instance of MockRepository
func NewMockRepository() *MockRepository {
	return &MockRepository{}
}

// WithDefaultBehavior sets up common default behaviors for the mock
func (m *MockRepository) WithDefaultBehavior() *MockRepository {
	// GetRootPath returns current directory by default
	m.On("GetRootPath", mock.Anything).Maybe().Return(".", nil)

	// IsGitRepository returns true by default
	m.On("IsGitRepository", mock.Anything, mock.Anything).Maybe().Return(true)

	// GetCurrentCommit returns a dummy commit hash
	m.On("GetCurrentCommit", mock.Anything, mock.Anything).Maybe().Return("abc123def456", nil)

	// GetRemoteURL returns a dummy URL
	m.On("GetRemoteURL", mock.Anything, mock.Anything, mock.Anything).Maybe().
		Return("https://github.com/user/repo.git", nil)

	// GetStatus returns a clean status by default
	m.On("GetStatus", mock.Anything, mock.Anything).Maybe().Return(&git.RepositoryStatus{
		IsClean:        true,
		ModifiedFiles:  []string{},
		UntrackedFiles: []string{},
		StagedFiles:    []string{},
	}, nil)

	return m
}

// GetRootPath mocks the GetRootPath method
func (m *MockRepository) GetRootPath(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

// IsGitRepository mocks the IsGitRepository method
func (m *MockRepository) IsGitRepository(ctx context.Context, path string) bool {
	args := m.Called(ctx, path)
	return args.Bool(0)
}

// GetCurrentCommit mocks the GetCurrentCommit method
func (m *MockRepository) GetCurrentCommit(ctx context.Context, path string) (string, error) {
	args := m.Called(ctx, path)
	return args.String(0), args.Error(1)
}

// GetRemoteURL mocks the GetRemoteURL method
func (m *MockRepository) GetRemoteURL(ctx context.Context, path string, remoteName string) (string, error) {
	args := m.Called(ctx, path, remoteName)
	return args.String(0), args.Error(1)
}

// GetStatus mocks the GetStatus method
func (m *MockRepository) GetStatus(ctx context.Context, path string) (*git.RepositoryStatus, error) {
	args := m.Called(ctx, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*git.RepositoryStatus), args.Error(1)
}

// Ensure MockRepository implements git.Repository interface
var _ git.Repository = (*MockRepository)(nil)
