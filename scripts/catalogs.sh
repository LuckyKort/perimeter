#!/bin/bash
# Renders the Homebrew cask and the winget manifests for a released version
# into build/catalogs/, with checksums taken from the release's SHA256SUMS.txt.
#
#   scripts/catalogs.sh 1.2.2
#
# Publishing them: copy Casks/kordon.rb to github.com/ef1m03/homebrew-tap, and
# open a pull request with manifests/ to github.com/microsoft/winget-pkgs.
set -euo pipefail
cd "$(dirname "$0")/.."
v="${1#v}"
base="https://github.com/ef1m03/kordon/releases/download/v$v"
sums="$(curl -fsSL "$base/SHA256SUMS.txt")"
sum() { echo "$sums" | awk -v n="$1" '$2 == n { print $1 }'; }
upper() { tr '[:lower:]' '[:upper:]'; }

mac="$(sum Kordon-macos.zip)"
x64="$(sum kordon-windows-amd64.exe | upper)"
arm="$(sum kordon-windows-arm64.exe | upper)"
[ -n "$mac" ] && [ -n "$x64" ] && [ -n "$arm" ] || { echo "release v$v has missing files" >&2; exit 1; }

out=build/catalogs
rm -rf "$out"
mkdir -p "$out/Casks"
cat > "$out/Casks/kordon.rb" <<EOF
cask "kordon" do
  version "$v"
  sha256 "$mac"

  url "https://github.com/ef1m03/kordon/releases/download/v#{version}/Kordon-macos.zip"
  name "Kordon"
  desc "Kill switch for apps: freezes them when the network or public IP changes"
  homepage "https://github.com/ef1m03/kordon"

  livecheck do
    url :url
    strategy :github_latest
  end

  # Kordon updates itself from its menu.
  auto_updates true
  depends_on macos: ">= :big_sur"

  app "Kordon.app"

  # Not notarized (that costs \$99/year); the build is attested on GitHub instead:
  #   gh attestation verify Kordon-macos.zip --repo ef1m03/kordon
  postflight do
    system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{appdir}/Kordon.app"]
  end

  uninstall launchctl: "dev.efimov.kordon",
            signal:    ["TERM", "dev.efimov.kordon"]

  zap trash: [
    "~/Library/Application Support/Kordon",
    "~/Library/LaunchAgents/dev.efimov.kordon.plist",
  ]
end
EOF

wdir="$out/manifests/e/EfimovDev/Kordon/$v"
mkdir -p "$wdir"
cat > "$wdir/EfimovDev.Kordon.yaml" <<EOF
# yaml-language-server: \$schema=https://aka.ms/winget-manifest.version.1.6.0.schema.json
PackageIdentifier: EfimovDev.Kordon
PackageVersion: $v
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.6.0
EOF
cat > "$wdir/EfimovDev.Kordon.installer.yaml" <<EOF
# yaml-language-server: \$schema=https://aka.ms/winget-manifest.installer.1.6.0.schema.json
PackageIdentifier: EfimovDev.Kordon
PackageVersion: $v
InstallerType: portable
Commands:
- kordon
UpgradeBehavior: install
ReleaseDate: $(date +%Y-%m-%d)
Installers:
- Architecture: x64
  InstallerUrl: $base/kordon-windows-amd64.exe
  InstallerSha256: $x64
- Architecture: arm64
  InstallerUrl: $base/kordon-windows-arm64.exe
  InstallerSha256: $arm
ManifestType: installer
ManifestVersion: 1.6.0
EOF
cat > "$wdir/EfimovDev.Kordon.locale.en-US.yaml" <<EOF
# yaml-language-server: \$schema=https://aka.ms/winget-manifest.defaultLocale.1.6.0.schema.json
PackageIdentifier: EfimovDev.Kordon
PackageVersion: $v
PackageLocale: en-US
Publisher: Efimov Dev
PublisherUrl: https://efimovdev.ru
PublisherSupportUrl: https://t.me/efimov_dev
PackageName: Kordon
PackageUrl: https://github.com/ef1m03/kordon
License: GPL-3.0
LicenseUrl: https://github.com/ef1m03/kordon/blob/main/LICENSE
Copyright: (c) Efimov Dev
ShortDescription: Kill switch for apps - freezes the apps you choose when your VPN drops or your public IP changes.
Description: |-
  Kordon watches your network and public IP. When the VPN drops, the IP changes or the internet goes away,
  it freezes the apps you chose (Claude, ChatGPT, Cursor, Telegram and others) within a fraction of a second,
  and resumes them when an allowed IP is back. Per-app rules, trusted networks, IPv6 and DNS leak checks,
  a tray icon with the country code and a settings panel. No admin rights, no telemetry, open source.
Moniker: kordon
Tags:
- kill-switch
- vpn
- privacy
- network
- ip
- tray
ReleaseNotesUrl: https://github.com/ef1m03/kordon/releases/tag/v$v
ManifestType: defaultLocale
ManifestVersion: 1.6.0
EOF
cat > "$wdir/EfimovDev.Kordon.locale.ru-RU.yaml" <<EOF
# yaml-language-server: \$schema=https://aka.ms/winget-manifest.locale.1.6.0.schema.json
PackageIdentifier: EfimovDev.Kordon
PackageVersion: $v
PackageLocale: ru-RU
Publisher: Efimov Dev
PackageName: Kordon
License: GPL-3.0
ShortDescription: Kill switch для приложений - замораживает выбранные программы, когда отваливается VPN или меняется IP.
ManifestType: locale
ManifestVersion: 1.6.0
EOF
echo "rendered $out for $v"
