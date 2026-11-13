//go:build !windows

package ui

import (
	"os"
	"path/filepath"
	"syscall"

	"perimeter/internal/app"
)

// PanelPagePath holds the page a running panel should switch to.
func PanelPagePath() string { return filepath.Join(app.Dir(), "panel-page") }

// focusPanel asks the running panel to come to the front (SIGUSR1) and to
// show page. false: it isn't running any more.
func focusPanel(p *os.Process, page string) bool {
	_ = os.WriteFile(PanelPagePath(), []byte(page), 0o600)
	return p.Signal(syscall.SIGUSR1) == nil
}
