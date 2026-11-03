#!/bin/bash
# Packs extension/ into dist/kordon-extension.zip for the Chrome Web Store
# (Chrome, Edge, Yandex Browser, Brave and other Chromium browsers).
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p dist
rm -f dist/kordon-extension.zip
(cd extension && zip -qr ../dist/kordon-extension.zip . -x '.*')
echo "built dist/kordon-extension.zip ($(python3 -c 'import json;print(json.load(open("extension/manifest.json"))["version"])'))"
