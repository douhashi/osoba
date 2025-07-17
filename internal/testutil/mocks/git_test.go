package mocks_test

import (
	"context"
	"errors"
	"testing"

	"github.com/douhashi/osoba/internal/git"
	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMockRepository_GetRootPath(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*mocks.MockRepository)
		wantPath  string
		wantErr   bool
	}{
		{
			name: "success",
			setupMock: func(m *mocks.MockRepository) {
				m.On("GetRootPath", mock.Anything).Return("/path/to/repo", nil)
			},
			wantPath: "/path/to/repo",
			wantErr:  false,
		},
		{
			name: "error",
			setupMock: func(m *mocks.MockRepository) {
				m.On("GetRootPath", mock.Anything).Return("", errors.New("not a git repository"))
			},
			wantPath: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockRepository()
			tt.setupMock(mockRepo)

			path, err := mockRepo.GetRootPath(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantPath, path)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestMockRepository_IsGitRepository(t *testing.T) {
	mockRepo := mocks.NewMockRepository()

	mockRepo.On("IsGitRepository", mock.Anything, "/valid/repo").Return(true)
	mockRepo.On("IsGitRepository", mock.Anything, "/not/repo").Return(false)

	assert.True(t, mockRepo.IsGitRepository(context.Background(), "/valid/repo"))
	assert.False(t, mockRepo.IsGitRepository(context.Background(), "/not/repo"))

	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetCurrentCommit(t *testing.T) {
	mockRepo := mocks.NewMockRepository()

	expectedCommit := "abc123def456"
	mockRepo.On("GetCurrentCommit", mock.Anything, "/path/to/repo").Return(expectedCommit, nil)

	commit, err := mockRepo.GetCurrentCommit(context.Background(), "/path/to/repo")

	assert.NoError(t, err)
	assert.Equal(t, expectedCommit, commit)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetRemoteURL(t *testing.T) {
	mockRepo := mocks.NewMockRepository()

	expectedURL := "https://github.com/user/repo.git"
	mockRepo.On("GetRemoteURL", mock.Anything, "/path/to/repo", "origin").Return(expectedURL, nil)

	url, err := mockRepo.GetRemoteURL(context.Background(), "/path/to/repo", "origin")

	assert.NoError(t, err)
	assert.Equal(t, expectedURL, url)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetStatus(t *testing.T) {
	mockRepo := mocks.NewMockRepository()

	expectedStatus := &git.RepositoryStatus{
		IsClean:        false,
		ModifiedFiles:  []string{"file1.go", "file2.go"},
		UntrackedFiles: []string{"new.txt"},
		StagedFiles:    []string{"staged.go"},
	}

	mockRepo.On("GetStatus", mock.Anything, "/path/to/repo").Return(expectedStatus, nil)

	status, err := mockRepo.GetStatus(context.Background(), "/path/to/repo")

	assert.NoError(t, err)
	assert.Equal(t, expectedStatus, status)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_WithDefaultBehavior(t *testing.T) {
	mockRepo := mocks.NewMockRepository().WithDefaultBehavior()

	// GetRootPath returns current directory by default
	path, err := mockRepo.GetRootPath(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, ".", path)

	// IsGitRepository returns true by default
	isGit := mockRepo.IsGitRepository(context.Background(), "/any/path")
	assert.True(t, isGit)

	// GetCurrentCommit returns a dummy commit
	commit, err := mockRepo.GetCurrentCommit(context.Background(), "/any/path")
	assert.NoError(t, err)
	assert.NotEmpty(t, commit)

	// GetRemoteURL returns a dummy URL
	url, err := mockRepo.GetRemoteURL(context.Background(), "/any/path", "origin")
	assert.NoError(t, err)
	assert.NotEmpty(t, url)

	// GetStatus returns a clean status
	status, err := mockRepo.GetStatus(context.Background(), "/any/path")
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.True(t, status.IsClean)
}

func TestMockRepository_ComplexScenario(t *testing.T) {
	mockRepo := mocks.NewMockRepository()

	// 複数の呼び出しをシミュレート
	mockRepo.On("GetRootPath", mock.Anything).Return("/workspace/project", nil).Once()
	mockRepo.On("IsGitRepository", mock.Anything, "/workspace/project").Return(true).Once()
	mockRepo.On("GetCurrentCommit", mock.Anything, "/workspace/project").Return("main123", nil).Once()
	mockRepo.On("GetStatus", mock.Anything, "/workspace/project").Return(&git.RepositoryStatus{
		IsClean:       true,
		ModifiedFiles: []string{},
	}, nil).Once()

	// 実行
	rootPath, err := mockRepo.GetRootPath(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "/workspace/project", rootPath)

	isGit := mockRepo.IsGitRepository(context.Background(), rootPath)
	assert.True(t, isGit)

	commit, err := mockRepo.GetCurrentCommit(context.Background(), rootPath)
	assert.NoError(t, err)
	assert.Equal(t, "main123", commit)

	status, err := mockRepo.GetStatus(context.Background(), rootPath)
	assert.NoError(t, err)
	assert.True(t, status.IsClean)
	assert.Empty(t, status.ModifiedFiles)

	mockRepo.AssertExpectations(t)
}
