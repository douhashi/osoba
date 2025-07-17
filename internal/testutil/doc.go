// Package testutil provides common test utilities, mocks, and builders for testing osoba components.
//
// This package is organized into the following sub-packages:
//
//   - mocks: Common mock implementations for interfaces used throughout the codebase
//   - builders: Test data builders using the builder pattern for creating test fixtures
//   - helpers: General test helper functions and utilities
//
// # Usage
//
// Import the specific sub-package you need:
//
//	import "github.com/douhashi/osoba/internal/testutil/mocks"
//	import "github.com/douhashi/osoba/internal/testutil/builders"
//
// # Example
//
// Using mocks:
//
//	mockGH := mocks.NewMockGitHubClient()
//	mockGH.On("GetIssue", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
//	    Return(&github.Issue{Number: 123}, nil)
//
// Using builders:
//
//	issue := builders.NewIssueBuilder().
//	    WithNumber(123).
//	    WithState("open").
//	    WithLabels([]string{"bug"}).
//	    Build()
package testutil
