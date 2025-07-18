package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorktreeManagerForIssue_GetWorktreePathForIssue(t *testing.T) {
	tests := []struct {
		name        string
		basePath    string
		issueNumber int
		want        string
	}{
		{
			name:        "通常のパス生成",
			basePath:    "/test/repo",
			issueNumber: 123,
			want:        "/test/repo/.git/osoba/worktrees/issue-123",
		},
		{
			name:        "別のIssue番号",
			basePath:    "/home/user/project",
			issueNumber: 456,
			want:        "/home/user/project/.git/osoba/worktrees/issue-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &worktreeManager{
				basePath: tt.basePath,
			}

			got := m.GetWorktreePathForIssue(tt.issueNumber)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWorktreeManager_generateBranchNameForIssue(t *testing.T) {
	tests := []struct {
		name        string
		issueNumber int
		want        string
	}{
		{
			name:        "Issue番号123",
			issueNumber: 123,
			want:        "osoba/#123",
		},
		{
			name:        "Issue番号456",
			issueNumber: 456,
			want:        "osoba/#456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &worktreeManager{}
			got := m.generateBranchNameForIssue(tt.issueNumber)
			assert.Equal(t, tt.want, got)
		})
	}
}
