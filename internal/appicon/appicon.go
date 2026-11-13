// Package appicon finds an installed app's bundle and renders its own icon,
// so the panel shows Claude, Cursor or Chrome as the system does. Nothing is
// bundled with Perimeter: the icons come from the apps on this computer.
package appicon

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ErrUnsupported: this OS has no icon source yet; the panel shows a letter.
var ErrUnsupported = errors.New("app icons are not supported on this OS")

// Bundle returns the .app bundle of an app: from a running process's path
// first (it is the copy actually in use), else by the bundle names in its
// match paths ("/Claude.app/") in the usual application folders. "" if none.
func Bundle(matchPaths, running []string) string {
	for _, p := range running {
		if i := strings.Index(p, ".app/"); i > 0 {
			return p[:i+4]
		}
	}
	home, _ := os.UserHomeDir()
	dirs := []string{"/Applications", filepath.Join(home, "Applications"), "/System/Applications"}
	for _, m := range matchPaths {
		name := strings.Trim(m, "/")
		if !strings.HasSuffix(name, ".app") || strings.Contains(name, "/") {
			continue
		}
		for _, d := range dirs {
			if st, err := os.Stat(filepath.Join(d, name)); err == nil && st.IsDir() {
				return filepath.Join(d, name)
			}
		}
	}
	return ""
}
