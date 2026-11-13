// mkicons draws the app icons: the macOS .iconset, the Windows .ico and a PNG
// for the README into the given folder, and with -assets also the committed
// PNGs of the panel and the browser extension.
//
//	go run ./tools/mkicons build/icons
//	go run ./tools/mkicons -assets
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"perimeter/internal/ui"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-assets" {
		assets()
		return
	}
	out := "build/icons"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	set := filepath.Join(out, "Perimeter.iconset")
	must(os.MkdirAll(set, 0o755))
	for _, n := range []int{16, 32, 128, 256, 512} {
		save(filepath.Join(set, fmt.Sprintf("icon_%dx%d.png", n, n)), n)
		save(filepath.Join(set, fmt.Sprintf("icon_%dx%d@2x.png", n, n)), n*2)
	}
	save(filepath.Join(out, "icon-256.png"), 256)
	must(os.WriteFile(filepath.Join(out, "perimeter.ico"),
		ui.ICO(ui.AppIcon(16), ui.AppIcon(32), ui.AppIcon(48), ui.AppIcon(256)), 0o644))
}

// assets redraws the PNGs kept in the repository; run it from the repository root.
func assets() {
	save("internal/ui/panel/icon.png", 128)
	for _, n := range []int{16, 32, 48, 128} {
		save(fmt.Sprintf("extension/icons/icon%d.png", n), n)
	}
	for _, n := range []int{16, 32} {
		write(fmt.Sprintf("extension/icons/off%d.png", n), ui.Logo(n, ui.Gray))
	}
}

func save(path string, n int) { write(path, ui.AppIcon(n)) }

func write(path string, img image.Image) {
	f, err := os.Create(path)
	must(err)
	defer f.Close()
	must(png.Encode(f, img))
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
