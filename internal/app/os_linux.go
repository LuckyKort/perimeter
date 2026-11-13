package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func Notify(title, body string) {
	_ = exec.Command("notify-send", "-a", "Perimeter", title, body).Start()
}

func OpenFile(path string) error { return exec.Command("xdg-open", path).Start() }
func OpenURL(u string) error     { return exec.Command("xdg-open", u).Start() }
func Reveal(path string) error   { return exec.Command("xdg-open", filepath.Dir(path)).Start() }

func SystemLanguage() string { return langFromEnv() }

func desktopFile() string {
	base, _ := os.UserConfigDir()
	return filepath.Join(base, "autostart", "perimeter.desktop")
}

func AutostartEnabled() bool {
	_, err := os.Stat(desktopFile())
	return err == nil
}

func SetAutostart(on bool) error {
	if !on {
		return os.Remove(desktopFile())
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(desktopFile()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(desktopFile(), []byte(fmt.Sprintf(
		"[Desktop Entry]\nType=Application\nName=Perimeter\nExec=%q\nX-GNOME-Autostart-enabled=true\n", exe)), 0o644)
}
