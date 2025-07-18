package tmux_test

import (
	"testing"

	"github.com/douhashi/osoba/internal/tmux"
	"github.com/stretchr/testify/assert"
)

func TestGetWindowNameForIssue(t *testing.T) {
	tests := []struct {
		name        string
		issueNumber int
		want        string
	}{
		{
			name:        "正常系: Issue番号123",
			issueNumber: 123,
			want:        "issue-123",
		},
		{
			name:        "正常系: Issue番号1",
			issueNumber: 1,
			want:        "issue-1",
		},
		{
			name:        "正常系: Issue番号999",
			issueNumber: 999,
			want:        "issue-999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := tmux.GetWindowNameForIssue(tt.issueNumber)

			// Assert
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseWindowNameForIssue(t *testing.T) {
	tests := []struct {
		name       string
		windowName string
		want       int
		wantErr    bool
	}{
		{
			name:       "正常系: issue-123",
			windowName: "issue-123",
			want:       123,
			wantErr:    false,
		},
		{
			name:       "正常系: issue-1",
			windowName: "issue-1",
			want:       1,
			wantErr:    false,
		},
		{
			name:       "異常系: 旧形式（フェーズ付き）",
			windowName: "123-plan",
			want:       0,
			wantErr:    true,
		},
		{
			name:       "異常系: 不正な形式",
			windowName: "invalid-format",
			want:       0,
			wantErr:    true,
		},
		{
			name:       "異常系: 数値以外",
			windowName: "issue-abc",
			want:       0,
			wantErr:    true,
		},
		{
			name:       "異常系: 空文字列",
			windowName: "",
			want:       0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got, err := tmux.ParseWindowNameForIssue(tt.windowName)

			// Assert
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestIsNewFormatIssueWindow(t *testing.T) {
	tests := []struct {
		name       string
		windowName string
		want       bool
	}{
		{
			name:       "正常系: issue-123",
			windowName: "issue-123",
			want:       true,
		},
		{
			name:       "正常系: issue-1",
			windowName: "issue-1",
			want:       true,
		},
		{
			name:       "異常系: 旧形式",
			windowName: "123-plan",
			want:       false,
		},
		{
			name:       "異常系: 通常のウィンドウ",
			windowName: "bash",
			want:       false,
		},
		{
			name:       "異常系: 空文字列",
			windowName: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := tmux.IsNewFormatIssueWindow(tt.windowName)

			// Assert
			assert.Equal(t, tt.want, got)
		})
	}
}
