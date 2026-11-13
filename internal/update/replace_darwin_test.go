package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBundleOf(t *testing.T) {
	if b, err := bundleOf("/Applications/Perimeter.app/Contents/MacOS/perimeter"); err != nil || b != "/Applications/Perimeter.app" {
		t.Fatalf("%q %v", b, err)
	}
	if _, err := bundleOf("/usr/local/bin/perimeter"); err == nil {
		t.Fatal("non-bundle accepted")
	}
}

// TestReplaceBundle swaps a fake installed Perimeter.app for a new one from a zip.
func TestReplaceBundle(t *testing.T) {
	root := t.TempDir()
	mk := func(dir, version string) {
		os.MkdirAll(filepath.Join(dir, "Perimeter.app", "Contents", "MacOS"), 0o755)
		os.WriteFile(filepath.Join(dir, "Perimeter.app", "Contents", "MacOS", "perimeter"), []byte(version), 0o755)
	}
	apps := filepath.Join(root, "Applications")
	mk(apps, "old")
	src := filepath.Join(root, "src")
	mk(src, "new")
	zip := filepath.Join(root, "Perimeter-macos.zip")
	if out, err := exec.Command("/usr/bin/ditto", "-c", "-k", "--keepParent", filepath.Join(src, "Perimeter.app"), zip).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	exe := filepath.Join(apps, "Perimeter.app", "Contents", "MacOS", "perimeter")
	restart, err := replace(exe, zip, root)
	if err != nil || restart == nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "new" {
		t.Fatalf("installed %q", b)
	}
	ents, _ := os.ReadDir(apps)
	if len(ents) != 1 {
		t.Fatalf("leftovers next to the app: %v", ents)
	}
}
