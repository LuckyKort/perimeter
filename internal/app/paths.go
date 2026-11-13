// Package app holds the OS glue: where files live, logging, notifications,
// autostart, opening files and links, and the loop that drives the guard.
package app

import (
	"os"
	"path/filepath"
	"runtime"
)

// Dir is where Perimeter keeps config.yaml, the log and the state file.
func Dir() string {
	if d := os.Getenv("PERIMETER_DIR"); d != "" {
		return d // a separate profile for development and screenshots
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	name := "Perimeter"
	if runtime.GOOS == "linux" {
		name = "perimeter"
	}
	return filepath.Join(base, name)
}

func ConfigPath() string { return filepath.Join(Dir(), "config.yaml") }
func LogPath() string    { return filepath.Join(Dir(), "perimeter.log") }
func StatePath() string  { return filepath.Join(Dir(), "state.json") }
func LockPath() string   { return filepath.Join(Dir(), "perimeter.lock") }

// Repo is this build's GitHub "owner/name": the About link and problem reports
// go there, and updates come from its releases. Set it at build time:
//
//	go build -ldflags "-X perimeter/internal/app.Repo=owner/name"
//
// Empty (the default) hides those links and turns updates off.
var Repo = ""

// RepoURL is Repo's page, or "" when the build has none.
func RepoURL() string {
	if Repo == "" {
		return ""
	}
	return "https://github.com/" + Repo
}
