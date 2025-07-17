// Package helpers provides general test helper functions and utilities.
//
// This package contains various helper functions that don't fit into the mocks or builders
// categories but are useful across multiple test suites.
//
// # Available Helpers
//
//   - Test fixtures loading
//   - Custom assertions
//   - Test environment setup/teardown
//   - Temporary file/directory management
//   - Test data generation utilities
//
// # Example
//
//	func TestWithTempDir(t *testing.T) {
//	    dir := helpers.CreateTempDir(t)
//	    defer helpers.CleanupTempDir(t, dir)
//
//	    // Use the temporary directory
//	    err := writeFile(filepath.Join(dir, "test.txt"), "content")
//	    // ...
//	}
package helpers
