package watcher

import (
	"strings"
	"testing"
	"time"
)

func TestIssueEvent(t *testing.T) {
	t.Run("IssueEventのフィールドが正しく設定される", func(t *testing.T) {
		now := time.Now()
		event := IssueEvent{
			Type:       LabelAdded,
			IssueID:    123,
			IssueTitle: "Test Issue",
			Owner:      "douhashi",
			Repo:       "osoba",
			FromLabel:  "",
			ToLabel:    "status:ready",
			Timestamp:  now,
		}

		if event.Type != LabelAdded {
			t.Errorf("Type = %v, want %v", event.Type, LabelAdded)
		}
		if event.IssueID != 123 {
			t.Errorf("IssueID = %v, want %v", event.IssueID, 123)
		}
		if event.IssueTitle != "Test Issue" {
			t.Errorf("IssueTitle = %v, want %v", event.IssueTitle, "Test Issue")
		}
		if event.Owner != "douhashi" {
			t.Errorf("Owner = %v, want %v", event.Owner, "douhashi")
		}
		if event.Repo != "osoba" {
			t.Errorf("Repo = %v, want %v", event.Repo, "osoba")
		}
		if event.FromLabel != "" {
			t.Errorf("FromLabel = %v, want empty", event.FromLabel)
		}
		if event.ToLabel != "status:ready" {
			t.Errorf("ToLabel = %v, want %v", event.ToLabel, "status:ready")
		}
		if !event.Timestamp.Equal(now) {
			t.Errorf("Timestamp = %v, want %v", event.Timestamp, now)
		}
	})

	t.Run("EventType定数が正しく定義される", func(t *testing.T) {
		// EventType定数のテスト
		if LabelAdded == "" {
			t.Error("LabelAdded should not be empty")
		}
		if LabelRemoved == "" {
			t.Error("LabelRemoved should not be empty")
		}
		if LabelChanged == "" {
			t.Error("LabelChanged should not be empty")
		}

		// 各定数が異なる値であることを確認
		if LabelAdded == LabelRemoved {
			t.Error("LabelAdded and LabelRemoved should be different")
		}
		if LabelAdded == LabelChanged {
			t.Error("LabelAdded and LabelChanged should be different")
		}
		if LabelRemoved == LabelChanged {
			t.Error("LabelRemoved and LabelChanged should be different")
		}
	})

	t.Run("String()メソッドが適切なフォーマットを返す", func(t *testing.T) {
		event := IssueEvent{
			Type:       LabelAdded,
			IssueID:    123,
			IssueTitle: "Test Issue",
			Owner:      "douhashi",
			Repo:       "osoba",
			ToLabel:    "status:ready",
			Timestamp:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		}

		str := event.String()
		if str == "" {
			t.Error("String() should not return empty")
		}

		// 必要な情報が含まれているか確認
		expectedContents := []string{
			"123",          // IssueID
			"Test Issue",   // IssueTitle
			"douhashi",     // Owner
			"osoba",        // Repo
			"status:ready", // ToLabel
			"label_added",  // Type
		}

		for _, expected := range expectedContents {
			if !containsString(str, expected) {
				t.Errorf("String() should contain %q, got %q", expected, str)
			}
		}
	})
}

func TestDetectLabelChanges(t *testing.T) {
	tests := []struct {
		name       string
		oldLabels  []string
		newLabels  []string
		wantEvents []IssueEvent
		issueID    int
		issueTitle string
		owner      string
		repo       string
	}{
		{
			name:       "ラベルが追加された場合",
			oldLabels:  []string{"bug"},
			newLabels:  []string{"bug", "status:ready"},
			issueID:    1,
			issueTitle: "Test Issue",
			owner:      "douhashi",
			repo:       "osoba",
			wantEvents: []IssueEvent{
				{
					Type:       LabelAdded,
					IssueID:    1,
					IssueTitle: "Test Issue",
					Owner:      "douhashi",
					Repo:       "osoba",
					ToLabel:    "status:ready",
				},
			},
		},
		{
			name:       "ラベルが削除された場合",
			oldLabels:  []string{"bug", "status:ready"},
			newLabels:  []string{"bug"},
			issueID:    2,
			issueTitle: "Another Issue",
			owner:      "douhashi",
			repo:       "osoba",
			wantEvents: []IssueEvent{
				{
					Type:       LabelRemoved,
					IssueID:    2,
					IssueTitle: "Another Issue",
					Owner:      "douhashi",
					Repo:       "osoba",
					FromLabel:  "status:ready",
				},
			},
		},
		{
			name:       "ラベルが変更された場合（status:プレフィックス）",
			oldLabels:  []string{"bug", "status:needs-plan"},
			newLabels:  []string{"bug", "status:ready"},
			issueID:    3,
			issueTitle: "Changed Issue",
			owner:      "douhashi",
			repo:       "osoba",
			wantEvents: []IssueEvent{
				{
					Type:       LabelChanged,
					IssueID:    3,
					IssueTitle: "Changed Issue",
					Owner:      "douhashi",
					Repo:       "osoba",
					FromLabel:  "status:needs-plan",
					ToLabel:    "status:ready",
				},
			},
		},
		{
			name:       "複数のラベルが同時に変更された場合",
			oldLabels:  []string{"bug"},
			newLabels:  []string{"enhancement", "status:ready", "priority:high"},
			issueID:    4,
			issueTitle: "Multiple Changes",
			owner:      "douhashi",
			repo:       "osoba",
			wantEvents: []IssueEvent{
				{
					Type:       LabelRemoved,
					IssueID:    4,
					IssueTitle: "Multiple Changes",
					Owner:      "douhashi",
					Repo:       "osoba",
					FromLabel:  "bug",
				},
				{
					Type:       LabelAdded,
					IssueID:    4,
					IssueTitle: "Multiple Changes",
					Owner:      "douhashi",
					Repo:       "osoba",
					ToLabel:    "enhancement",
				},
				{
					Type:       LabelAdded,
					IssueID:    4,
					IssueTitle: "Multiple Changes",
					Owner:      "douhashi",
					Repo:       "osoba",
					ToLabel:    "status:ready",
				},
				{
					Type:       LabelAdded,
					IssueID:    4,
					IssueTitle: "Multiple Changes",
					Owner:      "douhashi",
					Repo:       "osoba",
					ToLabel:    "priority:high",
				},
			},
		},
		{
			name:       "ラベルに変更がない場合",
			oldLabels:  []string{"bug", "status:ready"},
			newLabels:  []string{"bug", "status:ready"},
			issueID:    5,
			issueTitle: "No Changes",
			owner:      "douhashi",
			repo:       "osoba",
			wantEvents: []IssueEvent{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := DetectLabelChanges(tt.oldLabels, tt.newLabels)
			// テスト用にIssue情報を設定
			for i := range events {
				events[i].IssueID = tt.issueID
				events[i].IssueTitle = tt.issueTitle
				events[i].Owner = tt.owner
				events[i].Repo = tt.repo
				events[i].Timestamp = time.Time{} // テスト用に時刻を固定
			}

			if len(events) != len(tt.wantEvents) {
				t.Fatalf("got %d events, want %d events", len(events), len(tt.wantEvents))
			}

			// イベントの内容を確認（順序は問わない）
			for _, wantEvent := range tt.wantEvents {
				found := false
				for _, gotEvent := range events {
					if eventMatches(gotEvent, wantEvent) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected event not found: %+v", wantEvent)
				}
			}
		})
	}
}

// ヘルパー関数
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

func eventMatches(got, want IssueEvent) bool {
	return got.Type == want.Type &&
		got.IssueID == want.IssueID &&
		got.IssueTitle == want.IssueTitle &&
		got.Owner == want.Owner &&
		got.Repo == want.Repo &&
		got.FromLabel == want.FromLabel &&
		got.ToLabel == want.ToLabel
}
