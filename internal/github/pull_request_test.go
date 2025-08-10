package github

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetPullRequestForIssue(t *testing.T) {
	tests := []struct {
		name          string
		issueNumber   int
		mockGhOutput  string
		mockGhError   error
		expectedPR    *PullRequest
		expectedError bool
		errorContains string
	}{
		{
			name:        "正常系: Issue番号からPR情報を取得",
			issueNumber: 123,
			mockGhOutput: `[
				{
					"number": 456,
					"title": "feat: Add new feature",
					"state": "OPEN",
					"mergeable": "MERGEABLE",
					"isDraft": false,
					"headRefName": "feature/new-feature",
					"statusCheckRollup": {
						"state": "SUCCESS"
					}
				}
			]`,
			expectedPR: &PullRequest{
				Number:       456,
				Title:        "feat: Add new feature",
				State:        "OPEN",
				Mergeable:    "MERGEABLE",
				IsDraft:      false,
				HeadRefName:  "feature/new-feature",
				ChecksStatus: "SUCCESS",
			},
			expectedError: false,
		},
		{
			name:        "正常系: 複数のPRが存在する場合は最初のPRを返す",
			issueNumber: 123,
			mockGhOutput: `[
				{
					"number": 456,
					"title": "First PR",
					"state": "OPEN",
					"mergeable": "MERGEABLE"
				},
				{
					"number": 789,
					"title": "Second PR",
					"state": "OPEN",
					"mergeable": "MERGEABLE"
				}
			]`,
			expectedPR: &PullRequest{
				Number:    456,
				Title:     "First PR",
				State:     "OPEN",
				Mergeable: "MERGEABLE",
			},
			expectedError: false,
		},
		{
			name:          "正常系: PRが存在しない場合",
			issueNumber:   123,
			mockGhOutput:  "[]",
			expectedPR:    nil,
			expectedError: false,
		},
		{
			name:          "異常系: ghコマンドエラー",
			issueNumber:   123,
			mockGhError:   assert.AnError,
			expectedError: true,
			errorContains: "failed to list pull requests",
		},
		{
			name:          "異常系: JSONパースエラー",
			issueNumber:   123,
			mockGhOutput:  "invalid json",
			expectedError: true,
			errorContains: "failed to parse pull request",
		},
		{
			name:        "正常系: UNKNOWN mergeable status",
			issueNumber: 123,
			mockGhOutput: `[
				{
					"number": 456,
					"title": "feat: Add new feature",
					"state": "OPEN", 
					"mergeable": "UNKNOWN",
					"isDraft": false,
					"headRefName": "feature/new-feature",
					"statusCheckRollup": {
						"state": "PENDING"
					}
				}
			]`,
			expectedPR: &PullRequest{
				Number:       456,
				Title:        "feat: Add new feature",
				State:        "OPEN",
				Mergeable:    "UNKNOWN",
				IsDraft:      false,
				HeadRefName:  "feature/new-feature",
				ChecksStatus: "PENDING",
			},
			expectedError: false,
		},
		{
			name:        "正常系: CONFLICTING mergeable status",
			issueNumber: 123,
			mockGhOutput: `[
				{
					"number": 456,
					"title": "feat: Add new feature",
					"state": "OPEN",
					"mergeable": "CONFLICTING", 
					"isDraft": false,
					"headRefName": "feature/new-feature",
					"statusCheckRollup": {
						"state": "SUCCESS"
					}
				}
			]`,
			expectedPR: &PullRequest{
				Number:       456,
				Title:        "feat: Add new feature",
				State:        "OPEN",
				Mergeable:    "CONFLICTING",
				IsDraft:      false,
				HeadRefName:  "feature/new-feature",
				ChecksStatus: "SUCCESS",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &mockGhExecutor{
				output: tt.mockGhOutput,
				err:    tt.mockGhError,
			}

			client := &Client{
				owner:    "test-owner",
				repo:     "test-repo",
				executor: mockExecutor,
			}

			pr, err := client.GetPullRequestForIssue(context.Background(), tt.issueNumber)

			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedPR, pr)
			}

			// ghコマンドの引数を検証
			if tt.mockGhError == nil {
				assert.Contains(t, mockExecutor.lastCommand, "pr list")
				assert.Contains(t, mockExecutor.lastCommand, "--json")
				assert.Contains(t, mockExecutor.lastCommand, "--search")
				assert.Contains(t, mockExecutor.lastCommand, fmt.Sprintf("linked:%d", tt.issueNumber))
			}
		})
	}
}

func TestClient_MergePullRequest(t *testing.T) {
	tests := []struct {
		name          string
		prNumber      int
		mockGhOutput  string
		mockGhError   error
		expectedError bool
		errorContains string
	}{
		{
			name:          "正常系: PRのマージ成功",
			prNumber:      456,
			mockGhOutput:  "✓ Merged pull request #456 (feat: Add new feature)",
			expectedError: false,
		},
		{
			name:          "異常系: マージ失敗（コンフリクト）",
			prNumber:      456,
			mockGhError:   assert.AnError,
			mockGhOutput:  "X Pull request has conflicts",
			expectedError: true,
			errorContains: "failed to merge pull request",
		},
		{
			name:          "異常系: マージ失敗（CI失敗）",
			prNumber:      456,
			mockGhError:   assert.AnError,
			mockGhOutput:  "X Required status checks are not passing",
			expectedError: true,
			errorContains: "failed to merge pull request",
		},
		{
			name:          "異常系: PRが存在しない",
			prNumber:      999,
			mockGhError:   assert.AnError,
			mockGhOutput:  "could not find pull request",
			expectedError: true,
			errorContains: "failed to merge pull request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &mockGhExecutor{
				output: tt.mockGhOutput,
				err:    tt.mockGhError,
			}

			client := &Client{
				owner:    "test-owner",
				repo:     "test-repo",
				executor: mockExecutor,
			}

			err := client.MergePullRequest(context.Background(), tt.prNumber)

			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}

			// ghコマンドの引数を検証
			assert.Contains(t, mockExecutor.lastCommand, "pr merge")
			assert.Contains(t, mockExecutor.lastCommand, fmt.Sprintf("%d", tt.prNumber))
			assert.Contains(t, mockExecutor.lastCommand, "--squash")
			assert.Contains(t, mockExecutor.lastCommand, "--auto")
		})
	}
}

func TestClient_GetPullRequestStatus(t *testing.T) {
	tests := []struct {
		name          string
		prNumber      int
		mockGhOutput  string
		mockGhError   error
		expectedPR    *PullRequest
		expectedError bool
		errorContains string
	}{
		{
			name:     "正常系: PR状態を取得",
			prNumber: 456,
			mockGhOutput: `{
				"number": 456,
				"title": "feat: Add new feature",
				"state": "OPEN",
				"mergeable": "MERGEABLE",
				"isDraft": false,
				"headRefName": "feature/new-feature",
				"statusCheckRollup": {
					"state": "SUCCESS"
				}
			}`,
			expectedPR: &PullRequest{
				Number:       456,
				Title:        "feat: Add new feature",
				State:        "OPEN",
				Mergeable:    "MERGEABLE",
				IsDraft:      false,
				HeadRefName:  "feature/new-feature",
				ChecksStatus: "SUCCESS",
			},
			expectedError: false,
		},
		{
			name:     "正常系: ステータスチェックが進行中",
			prNumber: 456,
			mockGhOutput: `{
				"number": 456,
				"state": "OPEN",
				"mergeable": "UNKNOWN",
				"statusCheckRollup": {
					"state": "PENDING"
				}
			}`,
			expectedPR: &PullRequest{
				Number:       456,
				State:        "OPEN",
				Mergeable:    "UNKNOWN",
				ChecksStatus: "PENDING",
			},
			expectedError: false,
		},
		{
			name:          "異常系: ghコマンドエラー",
			prNumber:      456,
			mockGhError:   assert.AnError,
			expectedError: true,
			errorContains: "failed to get pull request",
		},
		{
			name:          "異常系: JSONパースエラー",
			prNumber:      456,
			mockGhOutput:  "invalid json",
			expectedError: true,
			errorContains: "failed to parse pull request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &mockGhExecutor{
				output: tt.mockGhOutput,
				err:    tt.mockGhError,
			}

			client := &Client{
				owner:    "test-owner",
				repo:     "test-repo",
				executor: mockExecutor,
			}

			pr, err := client.GetPullRequestStatus(context.Background(), tt.prNumber)

			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedPR, pr)
			}

			// ghコマンドの引数を検証
			if tt.mockGhError == nil {
				assert.Contains(t, mockExecutor.lastCommand, "pr view")
				assert.Contains(t, mockExecutor.lastCommand, fmt.Sprintf("%d", tt.prNumber))
				assert.Contains(t, mockExecutor.lastCommand, "--json")
			}
		})
	}
}

// TestGetPullRequestForIssueWithFallback tests the fallback mechanism for PR detection
func TestGetPullRequestForIssueWithFallback(t *testing.T) {
	t.Skip("Test to be implemented with fallback mechanism - this is a placeholder for TDD")

	// This test will verify:
	// 1. Primary linked: search works
	// 2. Fallback to branch name search when linked: search fails
	// 3. Proper logging of search attempts
	// 4. Error handling for both search methods
}

// TestGetPullRequestStatusWithRetry tests retry mechanism for PR status
func TestGetPullRequestStatusWithRetry(t *testing.T) {
	t.Skip("Test to be implemented with retry mechanism - this is a placeholder for TDD")

	// This test will verify:
	// 1. Retry when mergeable status is UNKNOWN
	// 2. Exponential backoff between retries
	// 3. Maximum retry attempts
	// 4. Proper logging of retry attempts
}
