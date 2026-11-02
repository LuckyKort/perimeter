package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceExe(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "kordon.exe")
	os.WriteFile(exe, []byte("old"), 0o755)
	fresh := filepath.Join(t.TempDir(), "kordon-windows-amd64.exe")
	os.WriteFile(fresh, []byte("new"), 0o755)
	if _, err := replace(exe, fresh, dir); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "new" {
		t.Fatalf("installed %q", b)
	}
	if b, _ := os.ReadFile(exe + ".old"); string(b) != "old" {
		t.Fatalf("old copy %q", b)
	}
}
