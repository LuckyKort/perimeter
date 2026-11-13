package window

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Open shows the panel as an Edge "app" window: no tabs or address bar, its own
// window in the taskbar. Edge ships with every Windows 10/11, so nothing extra
// is needed. A separate profile folder keeps it away from the user's own Edge.
func Open(url, _, _, _ string) error {
	edge := findEdge()
	if edge == "" {
		return errors.New("Microsoft Edge not found")
	}
	profile := filepath.Join(os.Getenv("LOCALAPPDATA"), "Perimeter", "panel-profile")
	cmd := exec.Command(edge, "--app="+url, fmt.Sprintf("--window-size=%d,%d", Width, Height),
		"--user-data-dir="+profile, "--no-first-run", "--disable-features=Translate")
	return cmd.Start()
}

func findEdge() string {
	for _, base := range []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles"), os.Getenv("LOCALAPPDATA")} {
		if base == "" {
			continue
		}
		p := filepath.Join(base, "Microsoft", "Edge", "Application", "msedge.exe")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("msedge.exe"); err == nil {
		return p
	}
	return ""
}
