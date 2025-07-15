package types

import (
	"testing"
)

func TestActionType(t *testing.T) {
	tests := []struct {
		name     string
		action   ActionType
		expected string
	}{
		{
			name:     "計画アクション",
			action:   ActionTypePlan,
			expected: "plan",
		},
		{
			name:     "実装アクション",
			action:   ActionTypeImplementation,
			expected: "implementation",
		},
		{
			name:     "レビューアクション",
			action:   ActionTypeReview,
			expected: "review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.action) != tt.expected {
				t.Errorf("ActionType = %v, want %v", string(tt.action), tt.expected)
			}
		})
	}
}

func TestBaseAction(t *testing.T) {
	t.Run("BaseActionの作成とActionType取得", func(t *testing.T) {
		ba := &BaseAction{Type: ActionTypePlan}

		if ba.ActionType() != "plan" {
			t.Errorf("ActionType() = %v, want %v", ba.ActionType(), "plan")
		}
	})
}
