// Package app holds the OS glue: where files live, logging, notifications,
// autostart, opening files and links, and the loop that drives the guard.
package app

import (
	"os"
	"path/filepath"
	"runtime"
)

// Dir is where Kordon keeps config.yaml, the log and the state file.
func Dir() string {
	if d := os.Getenv("KORDON_DIR"); d != "" {
		return d // a separate profile for development and screenshots
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	name := "Kordon"
	if runtime.GOOS == "linux" {
		name = "kordon"
	}
	return filepath.Join(base, name)
}

func ConfigPath() string { return filepath.Join(Dir(), "config.yaml") }
func LogPath() string    { return filepath.Join(Dir(), "kordon.log") }
func StatePath() string  { return filepath.Join(Dir(), "state.json") }
func LockPath() string   { return filepath.Join(Dir(), "kordon.lock") }

const (
	RepoURL = "https://github.com/ef1m03/kordon"
	// StudioURL is Kordon's page on the studio site; the tag tells it the visit came from the app.
	StudioURL   = "https://efimovdev.ru/kordon/?utm_source=kordon&utm_medium=app"
	TelegramURL = "https://t.me/efimov_dev"
)
