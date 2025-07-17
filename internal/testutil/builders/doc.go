// Package builders provides test data builders using the builder pattern for creating test fixtures.
//
// The builders in this package allow for easy creation of complex test data structures
// with sensible defaults and fluent APIs for customization.
//
// # Available Builders
//
//   - IssueBuilder: Creates github.Issue instances
//   - RepositoryBuilder: Creates github.Repository instances
//   - ConfigBuilder: Creates config.Config instances
//   - LabelBuilder: Creates github.Label instances
//
// # Example
//
//	func TestIssueProcessing(t *testing.T) {
//	    issue := NewIssueBuilder().
//	        WithNumber(123).
//	        WithState("open").
//	        WithTitle("Bug: Something is broken").
//	        WithLabels([]string{"bug", "priority:high"}).
//	        Build()
//
//	    // Use the issue in your test
//	    result := processIssue(issue)
//	    // ...
//	}
//
// # Best Practices
//
// 1. Builders should provide sensible defaults for all fields
// 2. Use method chaining for a fluent API
// 3. The Build() method should return an immutable copy
// 4. Consider providing preset methods like WithOpenState() for common configurations
package builders
