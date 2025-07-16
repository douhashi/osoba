package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/douhashi/osoba/internal/gh"
	"github.com/douhashi/osoba/internal/logger"
)

// GHLabelManager はghコマンドを使用するラベルマネージャー
type GHLabelManager struct {
	executor         gh.Executor
	logger           logger.Logger
	labelDefinitions map[string]LabelDefinition
	transitionRules  map[string]string
	maxRetries       int
	retryDelay       time.Duration
}

// NewGHLabelManager は新しいghコマンドベースのLabelManagerを作成する
func NewGHLabelManager(executor gh.Executor, logger logger.Logger, maxRetries int, retryDelay time.Duration) *GHLabelManager {
	lm := &GHLabelManager{
		executor:         executor,
		logger:           logger,
		labelDefinitions: make(map[string]LabelDefinition),
		transitionRules:  make(map[string]string),
		maxRetries:       maxRetries,
		retryDelay:       retryDelay,
	}

	// Initialize label definitions
	lm.initializeLabelDefinitions()

	// Initialize transition rules
	lm.initializeTransitionRules()

	return lm
}

// initializeLabelDefinitions sets up the label definitions
func (lm *GHLabelManager) initializeLabelDefinitions() {
	// Trigger labels
	lm.labelDefinitions["status:needs-plan"] = LabelDefinition{
		Name:        "status:needs-plan",
		Color:       "0075ca",
		Description: "Planning phase required",
	}
	lm.labelDefinitions["status:ready"] = LabelDefinition{
		Name:        "status:ready",
		Color:       "0E8A16",
		Description: "Ready for implementation",
	}
	lm.labelDefinitions["status:review-requested"] = LabelDefinition{
		Name:        "status:review-requested",
		Color:       "fbca04",
		Description: "Code review requested",
	}

	// Progress labels
	lm.labelDefinitions["status:planning"] = LabelDefinition{
		Name:        "status:planning",
		Color:       "c5def5",
		Description: "Currently in planning phase",
	}
	lm.labelDefinitions["status:implementing"] = LabelDefinition{
		Name:        "status:implementing",
		Color:       "bfd4f2",
		Description: "Currently being implemented",
	}
	lm.labelDefinitions["status:reviewing"] = LabelDefinition{
		Name:        "status:reviewing",
		Color:       "fef2c0",
		Description: "Currently under review",
	}
}

// initializeTransitionRules sets up the label transition rules
func (lm *GHLabelManager) initializeTransitionRules() {
	lm.transitionRules["status:needs-plan"] = "status:planning"
	lm.transitionRules["status:ready"] = "status:implementing"
	lm.transitionRules["status:review-requested"] = "status:reviewing"
}

// TransitionLabelWithRetry はリトライ機能付きでラベルを遷移させる
func (lm *GHLabelManager) TransitionLabelWithRetry(ctx context.Context, owner, repo string, issueNumber int) (bool, error) {
	transitioned, _, err := lm.TransitionLabelWithInfoWithRetry(ctx, owner, repo, issueNumber)
	return transitioned, err
}

// TransitionLabelWithInfoWithRetry はリトライ機能付きでラベルを遷移させ、詳細情報を返す
func (lm *GHLabelManager) TransitionLabelWithInfoWithRetry(ctx context.Context, owner, repo string, issueNumber int) (bool, *TransitionInfo, error) {
	var lastErr error
	for i := 0; i < lm.maxRetries; i++ {
		if i > 0 {
			if lm.logger != nil {
				lm.logger.Debug("Retrying label transition",
					"attempt", i+1,
					"issue", issueNumber,
					"delay", lm.retryDelay.String())
			}
			time.Sleep(lm.retryDelay)
		}

		transitioned, info, err := lm.transitionLabel(ctx, owner, repo, issueNumber)
		if err == nil {
			return transitioned, info, nil
		}

		lastErr = err
		if lm.logger != nil {
			lm.logger.Warn("Label transition failed",
				"attempt", i+1,
				"issue", issueNumber,
				"error", err)
		}
	}

	return false, nil, fmt.Errorf("failed after %d retries: %w", lm.maxRetries, lastErr)
}

// transitionLabel は実際のラベル遷移を実行する
func (lm *GHLabelManager) transitionLabel(ctx context.Context, owner, repo string, issueNumber int) (bool, *TransitionInfo, error) {
	// Issueの現在のラベルを取得
	labels, err := lm.getIssueLabels(ctx, owner, repo, issueNumber)
	if err != nil {
		return false, nil, fmt.Errorf("get issue labels: %w", err)
	}

	// 遷移可能なラベルを探す
	for _, label := range labels {
		if toLabel, exists := lm.transitionRules[label]; exists {
			// ラベルを削除
			if err := lm.removeLabel(ctx, owner, repo, issueNumber, label); err != nil {
				return false, nil, fmt.Errorf("remove label %s: %w", label, err)
			}

			// 新しいラベルを追加
			if err := lm.addLabel(ctx, owner, repo, issueNumber, toLabel); err != nil {
				return false, nil, fmt.Errorf("add label %s: %w", toLabel, err)
			}

			info := &TransitionInfo{
				TransitionFound: true,
				FromLabel:       label,
				ToLabel:         toLabel,
			}

			if lm.logger != nil {
				lm.logger.Info("Label transitioned",
					"issue", issueNumber,
					"from", label,
					"to", toLabel)
			}

			return true, info, nil
		}
	}

	info := &TransitionInfo{
		TransitionFound: false,
		CurrentLabels:   labels,
	}

	return false, info, nil
}

// getIssueLabels はIssueの現在のラベルを取得する
func (lm *GHLabelManager) getIssueLabels(ctx context.Context, owner, repo string, issueNumber int) ([]string, error) {
	args := []string{
		"issue", "view", strconv.Itoa(issueNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--json", "labels",
	}

	output, err := lm.executor.Execute(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("execute gh command: %w", err)
	}

	var issue struct {
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}

	if err := json.Unmarshal(output, &issue); err != nil {
		return nil, fmt.Errorf("parse issue response: %w", err)
	}

	var labels []string
	for _, label := range issue.Labels {
		labels = append(labels, label.Name)
	}

	return labels, nil
}

// addLabel はIssueにラベルを追加する
func (lm *GHLabelManager) addLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := []string{
		"issue", "edit", strconv.Itoa(issueNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--add-label", label,
	}

	if _, err := lm.executor.Execute(ctx, args); err != nil {
		return fmt.Errorf("add label: %w", err)
	}

	return nil
}

// removeLabel はIssueからラベルを削除する
func (lm *GHLabelManager) removeLabel(ctx context.Context, owner, repo string, issueNumber int, label string) error {
	args := []string{
		"issue", "edit", strconv.Itoa(issueNumber),
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--remove-label", label,
	}

	if _, err := lm.executor.Execute(ctx, args); err != nil {
		return fmt.Errorf("remove label: %w", err)
	}

	return nil
}

// EnsureLabelsExistWithRetry は必要なラベルが存在することを確認する
func (lm *GHLabelManager) EnsureLabelsExistWithRetry(ctx context.Context, owner, repo string) error {
	var lastErr error
	for i := 0; i < lm.maxRetries; i++ {
		if i > 0 {
			if lm.logger != nil {
				lm.logger.Debug("Retrying ensure labels exist",
					"attempt", i+1,
					"delay", lm.retryDelay.String())
			}
			time.Sleep(lm.retryDelay)
		}

		if err := lm.ensureLabelsExist(ctx, owner, repo); err == nil {
			return nil
		} else {
			lastErr = err
			if lm.logger != nil {
				lm.logger.Warn("Ensure labels exist failed",
					"attempt", i+1,
					"error", err)
			}
		}
	}

	return fmt.Errorf("failed after %d retries: %w", lm.maxRetries, lastErr)
}

// ensureLabelsExist は実際のラベル存在確認を実行する
func (lm *GHLabelManager) ensureLabelsExist(ctx context.Context, owner, repo string) error {
	// 既存のラベルを取得
	existingLabels, err := lm.listLabels(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("list labels: %w", err)
	}

	// 既存ラベルのマップを作成
	existing := make(map[string]bool)
	for _, label := range existingLabels {
		existing[label] = true
	}

	// 不足しているラベルを作成
	for name, def := range lm.labelDefinitions {
		if !existing[name] {
			if err := lm.createLabel(ctx, owner, repo, def); err != nil {
				return fmt.Errorf("create label %s: %w", name, err)
			}
			if lm.logger != nil {
				lm.logger.Info("Created label",
					"label", name,
					"color", def.Color,
					"description", def.Description)
			}
		}
	}

	return nil
}

// listLabels はリポジトリのラベル一覧を取得する
func (lm *GHLabelManager) listLabels(ctx context.Context, owner, repo string) ([]string, error) {
	args := []string{
		"label", "list",
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--json", "name",
	}

	output, err := lm.executor.Execute(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("execute gh command: %w", err)
	}

	var labels []struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal(output, &labels); err != nil {
		return nil, fmt.Errorf("parse labels response: %w", err)
	}

	var names []string
	for _, label := range labels {
		names = append(names, label.Name)
	}

	return names, nil
}

// createLabel は新しいラベルを作成する
func (lm *GHLabelManager) createLabel(ctx context.Context, owner, repo string, def LabelDefinition) error {
	args := []string{
		"label", "create", def.Name,
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--color", def.Color,
		"--description", def.Description,
	}

	if _, err := lm.executor.Execute(ctx, args); err != nil {
		return fmt.Errorf("create label: %w", err)
	}

	return nil
}

// TransitionInfo represents the result of a label transition operation
type TransitionInfo struct {
	TransitionFound bool
	FromLabel       string
	ToLabel         string
	CurrentLabels   []string
}

// Ensure GHLabelManager implements LabelManagerInterface
var _ LabelManagerInterface = (*GHLabelManager)(nil)
