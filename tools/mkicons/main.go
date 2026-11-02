// mkicons draws the app icons: the macOS .iconset, the Windows .ico and a PNG for the README.
//
//	go run ./tools/mkicons build/icons
package main

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"github.com/ef1m03/kordon/internal/ui"
)

func main() {
	out := "build/icons"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	set := filepath.Join(out, "Kordon.iconset")
	must(os.MkdirAll(set, 0o755))
	for _, n := range []int{16, 32, 128, 256, 512} {
		save(filepath.Join(set, fmt.Sprintf("icon_%dx%d.png", n, n)), n)
		save(filepath.Join(set, fmt.Sprintf("icon_%dx%d@2x.png", n, n)), n*2)
	}
	save(filepath.Join(out, "icon-256.png"), 256)
	must(os.WriteFile(filepath.Join(out, "kordon.ico"),
		ui.ICO(ui.AppIcon(16), ui.AppIcon(32), ui.AppIcon(48), ui.AppIcon(256)), 0o644))
}

func save(path string, n int) {
	f, err := os.Create(path)
	must(err)
	defer f.Close()
	must(png.Encode(f, ui.AppIcon(n)))
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
