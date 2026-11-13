package ui

import (
	"os"
	"testing"

	"perimeter/internal/app"
)

func TestWhatsNew(t *testing.T) {
	t.Setenv("PERIMETER_DIR", t.TempDir())
	defer func(c string) { changelog = c }(changelog)
	changelog = "# Changes\n\n## 1.2.2 - 03.11.2026\n\n- **Two.** Text.\n\n## 1.2.1 - 02.11.2026\n\n- **One.** Text.\n"

	// Fresh install: nothing is "new", and the version is remembered.
	if v, _ := newWhatsNew("1.2.2").get(); v != "" {
		t.Fatalf("fresh install shows %q", v)
	}
	if v, _ := newWhatsNew("1.2.2").get(); v != "" {
		t.Fatal("seen version shown again")
	}

	// An update from a version that didn't remember anything: state.json exists.
	_ = os.Remove(seenPath())
	_ = os.WriteFile(app.StatePath(), []byte("{}"), 0o644)
	w := newWhatsNew("1.2.2")
	if v, notes := w.get(); v != "1.2.2" || len(notes) == 0 {
		t.Fatalf("after update: %q %d notes", v, len(notes))
	}
	w.dismiss()
	if v, _ := newWhatsNew("1.2.2").get(); v != "" {
		t.Fatal("dismissed notes came back")
	}

	// The next update shows its own notes.
	if v, _ := newWhatsNew("1.2.1").get(); v != "1.2.1" {
		t.Fatalf("next version: %q", v)
	}
	// A dev build has no changelog section and shows nothing.
	if v, _ := newWhatsNew("dev").get(); v != "" {
		t.Fatal("dev build shows notes")
	}
}
