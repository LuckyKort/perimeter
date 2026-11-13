package ui

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"perimeter"
	"perimeter/internal/app"
	"perimeter/internal/update"
)

// UpdateInfo is what the tray window and the panel show about updates.
type UpdateInfo struct {
	Version    string        `json:"version"`
	Notes      []update.Note `json:"notes"`
	Page       string        `json:"page"`
	Installing bool          `json:"installing"`
	Done       int64         `json:"done"`
	Total      int64         `json:"total"`
}

// whatsNew remembers which version's notes the user has already seen, so after
// an update the tray window shows "Updated to X" once, until it is closed.
type whatsNew struct {
	mu      sync.Mutex
	version string
	notes   []update.Note
}

// changelog is the embedded CHANGELOG.md; tests swap it for their own text.
var changelog = perimeter.Changelog

func seenPath() string { return filepath.Join(app.Dir(), "seen-version") }

// newWhatsNew must run before Perimeter writes state.json for the first time:
// a state.json from an earlier run tells an update from a fresh install.
func newWhatsNew(version string) *whatsNew {
	w := &whatsNew{}
	notes := update.Notes(update.Section(changelog, version))
	if len(notes) == 0 {
		return w // dev build, or a version without a changelog section
	}
	b, _ := os.ReadFile(seenPath())
	seen := strings.TrimSpace(string(b))
	if seen == version {
		return w
	}
	if seen == "" {
		if _, err := os.Stat(app.StatePath()); err != nil {
			_ = os.WriteFile(seenPath(), []byte(version), 0o644) // fresh install: nothing is "new"
			return w
		}
	}
	w.version, w.notes = version, notes
	return w
}

func (w *whatsNew) get() (string, []update.Note) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.version, w.notes
}

func (w *whatsNew) dismiss() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.version != "" {
		_ = os.WriteFile(seenPath(), []byte(w.version), 0o644)
	}
	w.version, w.notes = "", nil
}

// CurrentNotes is this version's own changelog, for the panel's About page.
func CurrentNotes(version string) []update.Note {
	return update.Notes(update.Section(changelog, version))
}
