package ui

import (
	"os"
	"path/filepath"

	"perimeter/internal/app"
)

// PanelPagePath holds the page a running panel should switch to.
func PanelPagePath() string { return filepath.Join(app.Dir(), "panel-page") }

// focusPanel: Windows has no SIGUSR1, so a new panel window opens as before.
func focusPanel(*os.Process, string) bool { return false }
