package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// replace unpacks the binary next to the running one and renames it over it;
// Linux keeps the running file alive until the process exits.
func replace(exe, tgz, dir string) (func() error, error) {
	if out, err := exec.Command("tar", "-xzf", tgz, "-C", dir, "kordon").CombinedOutput(); err != nil {
		return nil, fmt.Errorf("распаковка: %v %s", err, out)
	}
	tmp := filepath.Join(filepath.Dir(exe), ".kordon-new")
	b, err := os.ReadFile(filepath.Join(dir, "kordon"))
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(tmp, b, 0o755); err != nil {
		return nil, fmt.Errorf("нет прав на запись в %s: %w", filepath.Dir(exe), err)
	}
	if err := os.Rename(tmp, exe); err != nil {
		os.Remove(tmp)
		return nil, err
	}
	return func() error {
		q := "'" + strings.ReplaceAll(exe, "'", `'\''`) + "'"
		cmd := exec.Command("/bin/sh", "-c", "sleep 1; nohup "+q+" >/dev/null 2>&1 &")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		return cmd.Start()
	}, nil
}

func Cleanup() {}
