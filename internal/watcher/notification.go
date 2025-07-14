package watcher

import (
	"sync"
)

// EventNotifier はイベントの通知を管理する
type EventNotifier struct {
	mu          sync.RWMutex
	buffer      chan IssueEvent
	subscribers []chan IssueEvent
	closed      bool
}

// NewEventNotifier は新しいEventNotifierを作成する
func NewEventNotifier(bufferSize int) *EventNotifier {
	return &EventNotifier{
		buffer:      make(chan IssueEvent, bufferSize),
		subscribers: make([]chan IssueEvent, 0),
	}
}

// Send はイベントを送信する（非ブロッキング）
func (n *EventNotifier) Send(event IssueEvent) bool {
	n.mu.RLock()
	if n.closed {
		n.mu.RUnlock()
		return false
	}
	n.mu.RUnlock()

	select {
	case n.buffer <- event:
		// 非同期でサブスクライバーに送信
		go n.broadcast(event)
		return true
	default:
		// バッファがフル
		return false
	}
}

// Broadcast はイベントを全サブスクライバーに送信する
func (n *EventNotifier) Broadcast(event IssueEvent) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.closed {
		return
	}

	for _, ch := range n.subscribers {
		select {
		case ch <- event:
			// 送信成功
		default:
			// チャネルがフル、スキップ
		}
	}
}

// Subscribe はイベントを受信するチャネルを取得する
func (n *EventNotifier) Subscribe() <-chan IssueEvent {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.closed {
		// クローズ済みの場合は、すぐにクローズされるチャネルを返す
		ch := make(chan IssueEvent)
		close(ch)
		return ch
	}

	ch := make(chan IssueEvent, 10) // 各サブスクライバーにバッファを持たせる
	n.subscribers = append(n.subscribers, ch)
	return ch
}

// Close は通知システムを終了する
func (n *EventNotifier) Close() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.closed {
		return
	}
	n.closed = true

	// バッファチャネルをクローズ
	close(n.buffer)

	// 全サブスクライバーのチャネルをクローズ
	for _, ch := range n.subscribers {
		close(ch)
	}
	n.subscribers = nil
}

// broadcast は内部的にサブスクライバーに送信する
func (n *EventNotifier) broadcast(event IssueEvent) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.closed {
		return
	}

	for _, ch := range n.subscribers {
		select {
		case ch <- event:
			// 送信成功
		default:
			// チャネルがフル、スキップ
		}
	}
}
