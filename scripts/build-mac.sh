#!/bin/bash
# Builds dist/Kordon.app (Apple Silicon + Intel) and dist/Kordon-macos.zip.
# Signed ad-hoc: free, and enough for install.sh, which downloads without the quarantine flag.
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${VERSION:-$(git describe --tags --always 2>/dev/null || echo dev)}"
VERSION="${VERSION#v}"
APP=dist/Kordon.app
rm -rf "$APP" dist/Kordon-macos.zip build/mac
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources" build/mac

for arch in arm64 amd64; do
  carch=$([ $arch = amd64 ] && echo x86_64 || echo arm64)
  # The panel window (WebKit) has C++ parts, so both compilers get the target arch.
  CC="clang -arch $carch" CXX="clang++ -arch $carch" CGO_CXXFLAGS="-Wno-deprecated-literal-operator" \
  CGO_ENABLED=1 GOOS=darwin GOARCH=$arch MACOSX_DEPLOYMENT_TARGET=11.0 \
    go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o build/mac/kordon-$arch ./cmd/kordon
done
lipo -create build/mac/kordon-arm64 build/mac/kordon-amd64 -output "$APP/Contents/MacOS/kordon"

go run ./tools/mkicons build/icons
iconutil -c icns build/icons/Kordon.iconset -o "$APP/Contents/Resources/Kordon.icns"

cat > "$APP/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>Kordon</string>
  <key>CFBundleDisplayName</key><string>Kordon</string>
  <key>CFBundleIdentifier</key><string>dev.efimov.kordon</string>
  <key>CFBundleExecutable</key><string>kordon</string>
  <key>CFBundleIconFile</key><string>Kordon</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>$VERSION</string>
  <key>CFBundleVersion</key><string>$VERSION</string>
  <key>LSMinimumSystemVersion</key><string>11.0</string>
  <key>LSUIElement</key><true/>
  <key>NSHumanReadableCopyright</key><string>ef1m03 · Efimov Dev · MIT</string>
</dict>
</plist>
EOF

codesign --force --deep --sign - "$APP"
(cd dist && ditto -c -k --keepParent Kordon.app Kordon-macos.zip)
echo "built $APP ($VERSION)"
