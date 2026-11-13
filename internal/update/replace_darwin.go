package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// bundleOf turns .../Perimeter.app/Contents/MacOS/perimeter into .../Perimeter.app.
func bundleOf(exe string) (string, error) {
	b := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
	if !strings.HasSuffix(b, ".app") {
		return "", fmt.Errorf("Perimeter запущен не из .app (%s) - обновите командой установки", exe)
	}
	return b, nil
}

// replace unpacks the new Perimeter.app next to the old one and swaps them with renames,
// so a failure halfway leaves the old version working.
func replace(exe, zip, dir string) (func() error, error) {
	bundle, err := bundleOf(exe)
	if err != nil {
		return nil, err
	}
	parent := filepath.Dir(bundle)
	stage, err := os.MkdirTemp(parent, ".perimeter-new-")
	if err != nil {
		return nil, fmt.Errorf("нет прав на запись в %s - обновите командой установки: %w", parent, err)
	}
	defer os.RemoveAll(stage)
	if out, err := exec.Command("/usr/bin/ditto", "-x", "-k", zip, stage).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("распаковка: %v %s", err, out)
	}
	fresh := filepath.Join(stage, "Perimeter.app")
	if _, err := os.Stat(filepath.Join(fresh, "Contents", "MacOS", "perimeter")); err != nil {
		return nil, fmt.Errorf("в архиве нет Perimeter.app")
	}
	_ = exec.Command("/usr/bin/xattr", "-dr", "com.apple.quarantine", fresh).Run()

	old := filepath.Join(stage, "Perimeter-old.app")
	if err := os.Rename(bundle, old); err != nil {
		return nil, err
	}
	if err := os.Rename(fresh, bundle); err != nil {
		_ = os.Rename(old, bundle) // put the old one back
		return nil, err
	}
	// The running process still has its files open, so removing the old bundle is safe.
	return func() error {
		// Started after we exit: the new instance needs our single-instance lock.
		// Start the binary itself, not `open`: while the old process is still exiting,
		// LaunchServices thinks Perimeter is running and `open` only "activates" it,
		// so nothing would start. The new one waits for our lock if we overlap.
		// Own session: launchd kills a job's whole process group when the job exits.
		bin := filepath.Join(bundle, "Contents", "MacOS", "perimeter")
		cmd := exec.Command("/bin/sh", "-c", "sleep 1; exec "+shellQuote(bin)+" >/dev/null 2>&1")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		return cmd.Start()
	}, nil
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func Cleanup() {}
