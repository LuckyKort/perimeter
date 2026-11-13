//go:build !darwin

package ui

func rightClick() bool { return false }

// yieldActivation: only macOS restricts which process may take the focus.
func yieldActivation() {}
