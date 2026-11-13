package appicon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBundleFromRunningProcess(t *testing.T) {
	got := Bundle(nil, []string{"/usr/bin/node", "/Applications/Google Chrome.app/Contents/Frameworks/x/Helpers/Google Chrome Helper.app/Contents/MacOS/Helper"})
	if got != "/Applications/Google Chrome.app" {
		t.Fatalf("got %q", got)
	}
}

func TestBundleByName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
	if err := os.MkdirAll(filepath.Join(home, "Applications", "Fake Perimeter Test.app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := Bundle([]string{"/Fake Perimeter Test.app/"}, nil); got != filepath.Join(home, "Applications", "Fake Perimeter Test.app") {
		t.Fatalf("got %q", got)
	}
	if got := Bundle([]string{`\AnthropicClaude\`, "claude"}, nil); got != "" {
		t.Fatalf("non-bundle paths matched %q", got)
	}
}
