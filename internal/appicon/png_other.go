//go:build !darwin

package appicon

// PNG: only macOS for now; elsewhere the panel shows the app's letter.
func PNG(string, int) ([]byte, error) { return nil, ErrUnsupported }
