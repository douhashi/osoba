package types

import (
	"testing"
	"time"
)

func TestIssuePhase(t *testing.T) {
	tests := []struct {
		name     string
		phase    IssuePhase
		expected string
	}{
		{
			name:     "計画フェーズ",
			phase:    IssueStatePlan,
			expected: "plan",
		},
		{
			name:     "実装フェーズ",
			phase:    IssueStateImplementation,
			expected: "implementation",
		},
		{
			name:     "レビューフェーズ",
			phase:    IssueStateReview,
			expected: "review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.phase) != tt.expected {
				t.Errorf("IssuePhase = %v, want %v", string(tt.phase), tt.expected)
			}
		})
	}
}

func TestIssueStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   IssueStatus
		expected string
	}{
		{
			name:     "保留中",
			status:   IssueStatusPending,
			expected: "pending",
		},
		{
			name:     "処理中",
			status:   IssueStatusProcessing,
			expected: "processing",
		},
		{
			name:     "完了",
			status:   IssueStatusCompleted,
			expected: "completed",
		},
		{
			name:     "失敗",
			status:   IssueStatusFailed,
			expected: "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("IssueStatus = %v, want %v", string(tt.status), tt.expected)
			}
		})
	}
}

func TestIssueState(t *testing.T) {
	t.Run("IssueStateの作成", func(t *testing.T) {
		now := time.Now()
		state := &IssueState{
			IssueNumber: 42,
			Phase:       IssueStatePlan,
			LastAction:  now,
			Status:      IssueStatusProcessing,
		}

		if state.IssueNumber != 42 {
			t.Errorf("IssueNumber = %v, want %v", state.IssueNumber, 42)
		}
		if state.Phase != IssueStatePlan {
			t.Errorf("Phase = %v, want %v", state.Phase, IssueStatePlan)
		}
		if !state.LastAction.Equal(now) {
			t.Errorf("LastAction = %v, want %v", state.LastAction, now)
		}
		if state.Status != IssueStatusProcessing {
			t.Errorf("Status = %v, want %v", state.Status, IssueStatusProcessing)
		}
	})
}
