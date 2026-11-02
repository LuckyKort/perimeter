package update

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// replace moves the running .exe aside (Windows allows renaming a running file,
// not overwriting it) and puts the new one in its place. The old file is
// removed by Cleanup on the next start.
func replace(exe, file, dir string) (func() error, error) {
	old := exe + ".old"
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		return nil, err
	}
	if err := moveFile(file, exe); err != nil {
		_ = os.Rename(old, exe)
		return nil, err
	}
	return func() error {
		// A hidden PowerShell waits for us to exit (we hold the single-instance lock), then starts the new version.
		ps := "Start-Sleep -Seconds 2; Start-Process -FilePath '" + strings.ReplaceAll(exe, "'", "''") + "'"
		cmd := exec.Command("powershell.exe", "-NoProfile", "-WindowStyle", "Hidden", "-Command", ps)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
		return cmd.Start()
	}, nil
}

// moveFile copies across volumes (%TEMP% may be on another drive than the app).
func moveFile(src, dst string) error {
	if os.Rename(src, dst) == nil {
		return nil
	}
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o755)
}

// Cleanup removes the previous version left by an update.
func Cleanup() {
	if exe, err := os.Executable(); err == nil {
		_ = os.Remove(exe + ".old")
	}
}
