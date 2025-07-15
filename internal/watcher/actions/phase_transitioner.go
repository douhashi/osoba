package actions

import (
	"context"
	"fmt"
	"log"
)

// PhaseTransitioner はフェーズ遷移を実行するインターフェース
type PhaseTransitioner interface {
	TransitionPhase(ctx context.Context, issueNumber int, phase string, from, to string) error
}

// GitHubClientInterface はGitHub操作のインターフェース
type GitHubClientInterface interface {
	CreateIssueComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error
	TransitionLabel(ctx context.Context, issueNumber int, from, to string) error
}

// ConfigProvider は設定を提供するインターフェース
type ConfigProvider interface {
	GetPhaseMessage(phase string) (string, bool)
}

// DefaultPhaseTransitioner はPhaseTransitionerのデフォルト実装
type DefaultPhaseTransitioner struct {
	owner        string
	repo         string
	githubClient GitHubClientInterface
	config       ConfigProvider
}

// NewPhaseTransitioner は新しいPhaseTransitionerを作成する
func NewPhaseTransitioner(owner, repo string, githubClient GitHubClientInterface, config ConfigProvider) PhaseTransitioner {
	return &DefaultPhaseTransitioner{
		owner:        owner,
		repo:         repo,
		githubClient: githubClient,
		config:       config,
	}
}

// TransitionPhase はフェーズ遷移を実行する
// 1. フェーズ開始コメントを投稿（失敗しても続行）
// 2. ラベル遷移を実行
func (t *DefaultPhaseTransitioner) TransitionPhase(ctx context.Context, issueNumber int, phase string, from, to string) error {
	// フェーズメッセージを取得
	message, found := t.config.GetPhaseMessage(phase)
	if found && message != "" {
		// コメント投稿（失敗してもエラーは無視）
		if err := t.githubClient.CreateIssueComment(ctx, t.owner, t.repo, issueNumber, message); err != nil {
			log.Printf("Failed to create comment for issue #%d: %v", issueNumber, err)
			// エラーは無視して処理を続行
		} else {
			log.Printf("Posted phase start comment for issue #%d: %s", issueNumber, message)
		}
	} else {
		log.Printf("No phase message found for phase: %s", phase)
	}

	// ラベル遷移
	if err := t.githubClient.TransitionLabel(ctx, issueNumber, from, to); err != nil {
		return fmt.Errorf("failed to transition label from %s to %s: %w", from, to, err)
	}

	log.Printf("Transitioned label for issue #%d: %s -> %s", issueNumber, from, to)
	return nil
}
