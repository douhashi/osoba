// Package mocks provides common mock implementations for interfaces used throughout the osoba codebase.
//
// These mocks are built using testify/mock and provide consistent behavior across all tests.
//
// # Available Mocks
//
//   - MockGitHubClient: Mock for github.Client interface
//   - MockLogger: Mock for log.Logger interface
//   - MockTmuxManager: Mock for tmux.Manager interface
//   - MockRepository: Mock for git.Repository interface
//   - MockClaudeExecutor: Mock for claude.Executor interface
//
// # Best Practices
//
// 1. Always use the factory functions (e.g., NewMockGitHubClient) to create mocks
// 2. Use WithDefaultBehavior() methods for common scenarios
// 3. Reset mocks between test cases when reusing them
// 4. Use mock.MatchedBy for complex argument matching
//
// # Example
//
//	func TestSomething(t *testing.T) {
//	    mockGH := NewMockGitHubClient()
//	    mockGH.On("GetIssue", mock.Anything, "owner", "repo", 123).
//	        Return(&github.Issue{Number: 123}, nil)
//
//	    // Use the mock in your test
//	    service := NewService(mockGH)
//	    // ...
//	}
package mocks
