package watcher

import (
	"testing"

	"github.com/douhashi/osoba/internal/github"
	"github.com/stretchr/testify/assert"
)

func TestShouldProcessIssue(t *testing.T) {
	tests := []struct {
		name           string
		issueLabels    []string
		expectedResult bool
		expectedReason string
	}{
		{
			name:           "トリガーラベル needs-plan があり、実行中ラベルがない場合は処理すべき",
			issueLabels:    []string{"status:needs-plan", "bug"},
			expectedResult: true,
			expectedReason: "Trigger label 'status:needs-plan' found without corresponding execution label",
		},
		{
			name:           "トリガーラベル ready があり、実行中ラベルがない場合は処理すべき",
			issueLabels:    []string{"status:ready", "enhancement"},
			expectedResult: true,
			expectedReason: "Trigger label 'status:ready' found without corresponding execution label",
		},
		{
			name:           "トリガーラベル review-requested があり、実行中ラベルがない場合は処理すべき",
			issueLabels:    []string{"status:review-requested"},
			expectedResult: true,
			expectedReason: "Trigger label 'status:review-requested' found without corresponding execution label",
		},
		{
			name:           "トリガーラベル needs-plan があるが、実行中ラベル planning もある場合は処理しない",
			issueLabels:    []string{"status:needs-plan", "status:planning"},
			expectedResult: false,
			expectedReason: "Execution label 'status:planning' already exists for trigger 'status:needs-plan'",
		},
		{
			name:           "トリガーラベル ready があるが、実行中ラベル implementing もある場合は処理しない",
			issueLabels:    []string{"status:ready", "status:implementing"},
			expectedResult: false,
			expectedReason: "Execution label 'status:implementing' already exists for trigger 'status:ready'",
		},
		{
			name:           "トリガーラベル review-requested があるが、実行中ラベル reviewing もある場合は処理しない",
			issueLabels:    []string{"status:review-requested", "status:reviewing"},
			expectedResult: false,
			expectedReason: "Execution label 'status:reviewing' already exists for trigger 'status:review-requested'",
		},
		{
			name:           "トリガーラベルがない場合は処理しない",
			issueLabels:    []string{"bug", "enhancement"},
			expectedResult: false,
			expectedReason: "No trigger labels found",
		},
		{
			name:           "ラベルが空の場合は処理しない",
			issueLabels:    []string{},
			expectedResult: false,
			expectedReason: "No trigger labels found",
		},
		{
			name:           "複数のトリガーラベルがある場合、最初に見つかったものを処理",
			issueLabels:    []string{"status:needs-plan", "status:ready"},
			expectedResult: true,
			expectedReason: "Trigger label 'status:needs-plan' found without corresponding execution label",
		},
		{
			name:           "異なるフェーズのトリガーと実行中ラベルがある場合は処理すべき",
			issueLabels:    []string{"status:ready", "status:planning"},
			expectedResult: true,
			expectedReason: "Trigger label 'status:ready' found without corresponding execution label",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			issue := createTestIssueWithLabels(tt.issueLabels)

			// Act
			shouldProcess, reason := ShouldProcessIssue(issue)

			// Assert
			assert.Equal(t, tt.expectedResult, shouldProcess, "処理判定が期待値と異なる")
			assert.Equal(t, tt.expectedReason, reason, "判定理由が期待値と異なる")
		})
	}
}

func TestGetTriggerLabelMapping(t *testing.T) {
	mapping := GetTriggerLabelMapping()

	// トリガーラベルと実行中ラベルの対応関係を確認
	assert.Equal(t, "status:planning", mapping["status:needs-plan"])
	assert.Equal(t, "status:implementing", mapping["status:ready"])
	assert.Equal(t, "status:reviewing", mapping["status:review-requested"])
}

// テスト用のヘルパー関数
func createTestIssueWithLabels(labelNames []string) *github.Issue {
	labels := make([]*github.Label, len(labelNames))
	for i, name := range labelNames {
		labelName := name
		labels[i] = &github.Label{Name: &labelName}
	}

	number := 1
	title := "Test Issue"
	return &github.Issue{
		Number: &number,
		Title:  &title,
		Labels: labels,
	}
}
