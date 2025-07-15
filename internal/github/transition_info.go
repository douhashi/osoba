package github

// TransitionInfo contains information about a label transition
type TransitionInfo struct {
	From string // The label that was removed
	To   string // The label that was added
}
