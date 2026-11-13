#!/bin/bash
# Packs extension/ into dist/perimeter-extension.zip for the Chrome Web Store
# (Chrome, Edge, Yandex Browser, Brave and other Chromium browsers).
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p dist
rm -f dist/perimeter-extension.zip
(cd extension && zip -qr ../dist/perimeter-extension.zip . -x '.*')
echo "built dist/perimeter-extension.zip ($(python3 -c 'import json;print(json.load(open("extension/manifest.json"))["version"])'))"
