#!/bin/bash
# Builds dist/kordon-windows-amd64.exe and -arm64.exe with the icon and a DPI-aware manifest.
# No cgo on Windows, so this cross-compiles from macOS or Linux.
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${VERSION:-$(git describe --tags --always 2>/dev/null || echo dev)}"
VERSION="${VERSION#v}"
mkdir -p dist
go run ./tools/mkicons build/icons

# go-winres writes rsrc_windows_*.syso next to main.go; the Go linker picks them up.
WINVER="$(echo "$VERSION" | grep -Eo '^[0-9]+(\.[0-9]+){0,3}' || echo 0.0.0)"
(cd cmd/kordon && go run github.com/tc-hib/go-winres@v0.3.3 simply \
  --icon ../../build/icons/icon-256.png --manifest gui \
  --product-name Kordon --file-description "Kordon - kill switch for apps" \
  --copyright "© Efimov Dev - efimovdev.ru" --original-filename kordon.exe \
  --product-version "$WINVER" --file-version "$WINVER" --arch amd64,arm64)

for arch in amd64 arm64; do
  CGO_ENABLED=0 GOOS=windows GOARCH=$arch \
    go build -trimpath -ldflags "-s -w -H windowsgui -X main.version=$VERSION" \
    -o dist/kordon-windows-$arch.exe ./cmd/kordon
done
rm -f cmd/kordon/rsrc_windows_*.syso
echo "built dist/kordon-windows-{amd64,arm64}.exe ($VERSION)"
