package github

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/douhashi/osoba/internal/logger"
)

// GHLabelManager はghコマンドを使用するラベルマネージャー
type GHLabelManager struct {
	logger           logger.Logger
	labelDefinitions map[string]LabelDefinition
	transitionRules  map[string]string
	maxRetries       int
	retryDelay       time.Duration
}

// NewGHLabelManager は新しいghコマンドベースのLabelManagerを作成する
func NewGHLabelManager(logger logger.Logger, maxRetries int, retryDelay time.Duration) *GHLabelManager {
	lm := &GHLabelManager{
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
	var lastInfo *TransitionInfo
	var transitioned bool

	operation := func() error {
		var err error
		transitioned, lastInfo, err = lm.transitionLabel(ctx, owner, repo, issueNumber)
		if err != nil {

			// Parse error and log appropriate details
			if ghErr, ok := err.(*GitHubError); ok {
				if lm.logger != nil {
					lm.logger.Warn("Label transition failed with GitHub error",
						"issue", issueNumber,
						"errorType", ghErr.Type.String(),
						"statusCode", ghErr.StatusCode,
						"message", ghErr.Message,
						"retryable", ghErr.IsRetryable())
				}
			} else {
				if lm.logger != nil {
					lm.logger.Warn("Label transition failed",
						"issue", issueNumber,
						"error", err)
				}
			}
			return err
		}
		return nil
	}

	// Use dynamic retry strategy based on error type
	err := operation()
	if err != nil {
		strategy := GetStrategyForError(err)
		// Override strategy with configured values if they are more conservative
		if lm.maxRetries < strategy.MaxAttempts {
			strategy.MaxAttempts = lm.maxRetries
		}
		if lm.retryDelay > strategy.InitialDelay {
			strategy.InitialDelay = lm.retryDelay
		}

		if lm.logger != nil {
			lm.logger.Debug("Using retry strategy",
				"maxAttempts", strategy.MaxAttempts,
				"initialDelay", strategy.InitialDelay,
				"errorType", fmt.Sprintf("%T", err))
		}

		err = RetryWithStrategy(ctx, strategy, operation)
	}

	if err != nil {
		return false, nil, fmt.Errorf("label transition failed: %w", err)
	}

	return transitioned, lastInfo, nil
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

	output, err := lm.executeGHCommand(ctx, args...)
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

	if _, err := lm.executeGHCommand(ctx, args...); err != nil {
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

	if _, err := lm.executeGHCommand(ctx, args...); err != nil {
		return fmt.Errorf("remove label: %w", err)
	}

	return nil
}

// EnsureLabelsExistWithRetry は必要なラベルが存在することを確認する
func (lm *GHLabelManager) EnsureLabelsExistWithRetry(ctx context.Context, owner, repo string) error {
	operation := func() error {
		err := lm.ensureLabelsExist(ctx, owner, repo)
		if err != nil {
			// Parse error and log appropriate details
			if ghErr, ok := err.(*GitHubError); ok {
				if lm.logger != nil {
					lm.logger.Warn("Ensure labels exist failed with GitHub error",
						"repo", fmt.Sprintf("%s/%s", owner, repo),
						"errorType", ghErr.Type.String(),
						"statusCode", ghErr.StatusCode,
						"message", ghErr.Message,
						"retryable", ghErr.IsRetryable())
				}
			} else {
				if lm.logger != nil {
					lm.logger.Warn("Ensure labels exist failed",
						"repo", fmt.Sprintf("%s/%s", owner, repo),
						"error", err)
				}
			}
		}
		return err
	}

	// Use dynamic retry strategy based on error type
	err := operation()
	if err != nil {
		strategy := GetStrategyForError(err)
		// Override strategy with configured values if they are more conservative
		if lm.maxRetries < strategy.MaxAttempts {
			strategy.MaxAttempts = lm.maxRetries
		}
		if lm.retryDelay > strategy.InitialDelay {
			strategy.InitialDelay = lm.retryDelay
		}

		if lm.logger != nil {
			lm.logger.Debug("Using retry strategy for ensure labels",
				"maxAttempts", strategy.MaxAttempts,
				"initialDelay", strategy.InitialDelay,
				"errorType", fmt.Sprintf("%T", err))
		}

		err = RetryWithStrategy(ctx, strategy, operation)
	}

	if err != nil {
		return fmt.Errorf("ensure labels exist failed: %w", err)
	}

	return nil
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

	output, err := lm.executeGHCommand(ctx, args...)
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

	if _, err := lm.executeGHCommand(ctx, args...); err != nil {
		return fmt.Errorf("create label: %w", err)
	}

	return nil
}

// executeGHCommand はghコマンドを実行する
func (lm *GHLabelManager) executeGHCommand(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Parse the error output to create a structured GitHubError
		ghErr := ParseGHError(string(output), err)
		return nil, ghErr
	}
	return output, nil
}

// Ensure GHLabelManager implements LabelManagerInterface
var _ LabelManagerInterface = (*GHLabelManager)(nil)
