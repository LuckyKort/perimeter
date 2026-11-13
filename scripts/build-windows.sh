#!/bin/bash
# Builds dist/perimeter-windows-amd64.exe and -arm64.exe with the icon and a DPI-aware manifest.
# No cgo on Windows, so this cross-compiles from macOS or Linux.
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${VERSION:-$(git describe --tags --always 2>/dev/null || echo dev)}"
VERSION="${VERSION#v}"
# REPO=owner/name: another repository for updates, the About link and problem
# reports (the default is in internal/app/paths.go). REPO= with no value turns them off.
LDREPO=""
if [ "${REPO+set}" = set ]; then LDREPO="-X perimeter/internal/app.Repo=$REPO"; fi
mkdir -p dist
go run ./tools/mkicons build/icons

# go-winres writes rsrc_windows_*.syso next to main.go; the Go linker picks them up.
WINVER="$(echo "$VERSION" | grep -Eo '^[0-9]+(\.[0-9]+){0,3}' || echo 0.0.0)"
(cd cmd/perimeter && go run github.com/tc-hib/go-winres@v0.3.3 simply \
  --icon ../../build/icons/icon-256.png --manifest gui \
  --product-name Perimeter --file-description "Perimeter - kill switch for apps" \
  --copyright "Perimeter, GPL-3.0. Based on Kordon © Efimov Dev" --original-filename perimeter.exe \
  --product-version "$WINVER" --file-version "$WINVER" --arch amd64,arm64)

for arch in amd64 arm64; do
  CGO_ENABLED=0 GOOS=windows GOARCH=$arch \
    go build -trimpath -ldflags "-s -w -H windowsgui -X main.version=$VERSION $LDREPO" \
    -o dist/perimeter-windows-$arch.exe ./cmd/perimeter
done
rm -f cmd/perimeter/rsrc_windows_*.syso
echo "built dist/perimeter-windows-{amd64,arm64}.exe ($VERSION)"
