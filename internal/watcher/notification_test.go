package watcher

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/douhashi/osoba/internal/github"
	"github.com/douhashi/osoba/internal/testutil/builders"
	"github.com/douhashi/osoba/internal/testutil/mocks"
	"github.com/stretchr/testify/mock"
)

func TestEventNotifier(t *testing.T) {
	t.Run("イベントを正しく送受信できる", func(t *testing.T) {
		notifier := NewEventNotifier(10) // バッファサイズ10

		// イベントを送信
		event1 := IssueEvent{
			Type:       LabelAdded,
			IssueID:    1,
			IssueTitle: "Test Issue 1",
			Owner:      "douhashi",
			Repo:       "osoba",
			ToLabel:    "status:ready",
			Timestamp:  time.Now(),
		}

		event2 := IssueEvent{
			Type:       LabelRemoved,
			IssueID:    2,
			IssueTitle: "Test Issue 2",
			Owner:      "douhashi",
			Repo:       "osoba",
			FromLabel:  "status:needs-plan",
			Timestamp:  time.Now(),
		}

		// イベントを受信するためのサブスクライバーを先に作成
		ch := notifier.Subscribe()

		// 非ブロッキングで送信
		if !notifier.Send(event1) {
			t.Error("Send() should succeed for event1")
		}
		if !notifier.Send(event2) {
			t.Error("Send() should succeed for event2")
		}

		// 受信したイベントのIDを記録
		receivedIDs := make(map[int]bool)

		for i := 0; i < 2; i++ {
			select {
			case received := <-ch:
				receivedIDs[received.IssueID] = true
			case <-time.After(100 * time.Millisecond):
				t.Errorf("Timeout waiting for event %d", i+1)
			}
		}

		// 両方のイベントが受信されたことを確認
		if !receivedIDs[event1.IssueID] {
			t.Error("Event1 was not received")
		}
		if !receivedIDs[event2.IssueID] {
			t.Error("Event2 was not received")
		}
	})

	t.Run("バッファがフルの場合はドロップされる", func(t *testing.T) {
		notifier := NewEventNotifier(2) // 小さなバッファ

		// バッファを埋める
		event1 := IssueEvent{IssueID: 1}
		event2 := IssueEvent{IssueID: 2}
		event3 := IssueEvent{IssueID: 3}

		if !notifier.Send(event1) {
			t.Error("Send() should succeed for event1")
		}
		if !notifier.Send(event2) {
			t.Error("Send() should succeed for event2")
		}

		// バッファがフルなのでfalseを返す
		if notifier.Send(event3) {
			t.Error("Send() should return false when buffer is full")
		}
	})

	t.Run("複数のサブスクライバーが同じイベントを受信", func(t *testing.T) {
		notifier := NewEventNotifier(10)

		// 複数のサブスクライバー
		ch1 := notifier.Subscribe()
		ch2 := notifier.Subscribe()
		ch3 := notifier.Subscribe()

		event := IssueEvent{
			Type:       LabelAdded,
			IssueID:    123,
			IssueTitle: "Broadcast Test",
			Timestamp:  time.Now(),
		}

		// ブロードキャスト
		notifier.Broadcast(event)

		// 全員が受信することを確認
		channels := []<-chan IssueEvent{ch1, ch2, ch3}
		for i, ch := range channels {
			select {
			case received := <-ch:
				if received.IssueID != event.IssueID {
					t.Errorf("Subscriber %d: received IssueID = %v, want %v", i+1, received.IssueID, event.IssueID)
				}
			case <-time.After(100 * time.Millisecond):
				t.Errorf("Subscriber %d: timeout waiting for event", i+1)
			}
		}
	})

	t.Run("Closeで全チャネルがクローズされる", func(t *testing.T) {
		notifier := NewEventNotifier(10)

		ch1 := notifier.Subscribe()
		ch2 := notifier.Subscribe()

		// クローズ
		notifier.Close()

		// チャネルがクローズされていることを確認
		_, ok1 := <-ch1
		if ok1 {
			t.Error("ch1 should be closed")
		}

		_, ok2 := <-ch2
		if ok2 {
			t.Error("ch2 should be closed")
		}

		// クローズ後のSendはfalseを返す
		if notifier.Send(IssueEvent{}) {
			t.Error("Send() should return false after Close()")
		}
	})
}

func TestWatcherWithEventNotification(t *testing.T) {
	t.Run("IssueWatcherからイベント通知を受信", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockClient := mocks.NewMockGitHubClient()

		// モックの設定
		testIssues := []*github.Issue{
			builders.NewIssueBuilder().
				WithNumber(1).
				WithTitle("Test Issue 1").
				WithLabels([]string{"status:ready"}).
				Build(),
		}
		mockClient.On("ListIssuesByLabels", mock.Anything, "douhashi", "osoba", []string{"status:ready"}).
			Return(testIssues, nil).Maybe()
		mockClient.On("GetRateLimit", mock.Anything).
			Return(builders.NewRateLimitsBuilder().Build(), nil).Maybe()

		watcher, err := NewIssueWatcher(mockClient, "douhashi", "osoba", "test-session", []string{"status:ready"}, 5*time.Second, NewMockLogger())
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		// イベント通知を有効化
		notifier := NewEventNotifier(10)
		watcher.SetEventNotifier(notifier)

		// ポーリング間隔を短く
		// テスト用に短いポーリング間隔を設定
		watcher.SetPollIntervalForTest(100 * time.Millisecond)

		// イベントを受信するゴルーチン
		eventCh := notifier.Subscribe()
		received := make(chan IssueEvent, 10)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for event := range eventCh {
				received <- event
			}
		}()

		// 監視開始
		go watcher.Start(ctx, func(issue *github.Issue) {
			// コールバックは空でOK（イベント通知でテスト）
		})

		// イベントを待つ
		select {
		case event := <-received:
			if event.IssueID != 1 {
				t.Errorf("Event IssueID = %v, want 1", event.IssueID)
			}
			if event.Type != "" {
				// 新規検出の場合、イベントタイプは設定されない可能性がある
				t.Logf("Event type: %v", event.Type)
			}
		case <-time.After(500 * time.Millisecond):
			t.Error("Timeout waiting for event")
		}

		// クリーンアップ
		cancel()
		notifier.Close()
		wg.Wait()
	})
}

// MockGitHubClientにラベル変更をシミュレートする機能を追加
func TestLabelChangeEventNotification(t *testing.T) {
	t.Run("ラベル変更時にイベントが通知される", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockClient := mocks.NewMockGitHubClient()

		// 初期状態のIssue
		initialIssues := []*github.Issue{
			builders.NewIssueBuilder().
				WithNumber(1).
				WithTitle("Test Issue").
				WithLabels([]string{"bug"}).
				Build(),
		}

		// 更新後の状態（ラベルが追加される）
		updatedIssues := []*github.Issue{
			builders.NewIssueBuilder().
				WithNumber(1).
				WithTitle("Test Issue").
				WithLabels([]string{"bug", "status:ready"}).
				Build(),
		}

		// 最初は初期状態、その後は更新後の状態を返す
		mockClient.On("ListIssuesByLabels", mock.Anything, "douhashi", "osoba", []string{"bug", "status:ready"}).
			Return(initialIssues, nil).Once()
		mockClient.On("ListIssuesByLabels", mock.Anything, "douhashi", "osoba", []string{"bug", "status:ready"}).
			Return(updatedIssues, nil).Maybe()
		mockClient.On("GetRateLimit", mock.Anything).
			Return(builders.NewRateLimitsBuilder().Build(), nil).Maybe()

		watcher, err := NewIssueWatcherWithLabelTracking(mockClient, "douhashi", "osoba", "test-session", []string{"bug", "status:ready"}, 5*time.Second, NewMockLogger())
		if err != nil {
			t.Fatalf("failed to create watcher: %v", err)
		}

		// イベント通知を有効化
		notifier := NewEventNotifier(10)
		watcher.SetEventNotifier(notifier)
		watcher.EnableLabelChangeTracking(true)

		// ポーリング間隔を短く
		// テスト用に短いポーリング間隔を設定
		watcher.SetPollIntervalForTest(100 * time.Millisecond)

		// イベントを受信
		eventCh := notifier.Subscribe()
		received := make(chan IssueEvent, 10)

		go func() {
			for event := range eventCh {
				if event.Type == LabelAdded || event.Type == LabelRemoved || event.Type == LabelChanged {
					received <- event
				}
			}
		}()

		// 監視開始
		go watcher.Start(ctx, func(issue *github.Issue) {})

		// ラベル追加イベントを待つ
		select {
		case event := <-received:
			if event.Type != LabelAdded {
				t.Errorf("Event Type = %v, want %v", event.Type, LabelAdded)
			}
			if event.ToLabel != "status:ready" {
				t.Errorf("Event ToLabel = %v, want status:ready", event.ToLabel)
			}
			if event.IssueID != 1 {
				t.Errorf("Event IssueID = %v, want 1", event.IssueID)
			}
		case <-time.After(500 * time.Millisecond):
			t.Error("Timeout waiting for label change event")
		}

		// クリーンアップ
		cancel()
		notifier.Close()
	})
}
