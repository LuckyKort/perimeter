#!/bin/bash
# Builds dist/Perimeter.app (Apple Silicon + Intel) and dist/Perimeter-macos.zip.
# Signed ad-hoc: free, and enough for install.sh, which downloads without the quarantine flag.
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${VERSION:-$(git describe --tags --always 2>/dev/null || echo dev)}"
VERSION="${VERSION#v}"
# REPO=owner/name turns on the About link, problem reports and updates from its releases.
REPO="${REPO:-}"
APP=dist/Perimeter.app
rm -rf "$APP" dist/Perimeter-macos.zip build/mac
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources" build/mac

for arch in arm64 amd64; do
  carch=$([ $arch = amd64 ] && echo x86_64 || echo arm64)
  # The panel window (WebKit) has C++ parts, so both compilers get the target arch.
  CC="clang -arch $carch" CXX="clang++ -arch $carch" CGO_CXXFLAGS="-Wno-deprecated-literal-operator" \
  CGO_ENABLED=1 GOOS=darwin GOARCH=$arch MACOSX_DEPLOYMENT_TARGET=11.0 \
    go build -trimpath -ldflags "-s -w -X main.version=$VERSION -X perimeter/internal/app.Repo=$REPO" -o build/mac/perimeter-$arch ./cmd/perimeter
done
lipo -create build/mac/perimeter-arm64 build/mac/perimeter-amd64 -output "$APP/Contents/MacOS/perimeter"

go run ./tools/mkicons build/icons
iconutil -c icns build/icons/Perimeter.iconset -o "$APP/Contents/Resources/Perimeter.icns"

cat > "$APP/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>Perimeter</string>
  <key>CFBundleDisplayName</key><string>Perimeter</string>
  <key>CFBundleIdentifier</key><string>dev.perimeter.app</string>
  <key>CFBundleExecutable</key><string>perimeter</string>
  <key>CFBundleIconFile</key><string>Perimeter</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>$VERSION</string>
  <key>CFBundleVersion</key><string>$VERSION</string>
  <key>LSMinimumSystemVersion</key><string>11.0</string>
  <key>LSUIElement</key><true/>
  <key>NSHumanReadableCopyright</key><string>Perimeter, GPL-3.0. Based on Kordon © Efimov Dev</string>
</dict>
</plist>
EOF

codesign --force --deep --sign - "$APP"
(cd dist && ditto -c -k --keepParent Perimeter.app Perimeter-macos.zip)
echo "built $APP ($VERSION)"
