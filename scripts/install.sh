#!/bin/sh
# Kordon installer for macOS and Linux.
#
#   curl -fsSL https://raw.githubusercontent.com/ef1m03/kordon/main/scripts/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/ef1m03/kordon/main/scripts/install.sh | sh -s uninstall
#
# On macOS the app is downloaded with curl, so it carries no quarantine flag and
# Gatekeeper doesn't block it although it isn't notarized (that costs $99/year).
set -eu

REPO="ef1m03/kordon"
BASE="https://github.com/$REPO/releases/latest/download"
ACTION="${1:-install}"

say() { printf '\033[1m%s\033[0m\n' "$*"; }

stop_running() {
  # Quitting through the menu would un-freeze apps; on update Kordon does it at its next start.
  pkill -x kordon 2>/dev/null || true
  sleep 1
}

mac() {
  dest="/Applications"
  [ -w "$dest" ] || dest="$HOME/Applications"
  if [ "$ACTION" = "uninstall" ]; then
    stop_running
    rm -rf "$dest/Kordon.app" "$HOME/Applications/Kordon.app"
    rm -f "$HOME/Library/LaunchAgents/dev.efimov.kordon.plist"
    say "Kordon удалён. Настройки остались в ~/Library/Application Support/Kordon - удалите папку, если не нужны."
    return
  fi
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT
  say "Скачиваю Kordon…"
  curl -fsSL "$BASE/Kordon-macos.zip" -o "$tmp/Kordon.zip"
  stop_running
  mkdir -p "$dest"
  rm -rf "$dest/Kordon.app"
  ditto -x -k "$tmp/Kordon.zip" "$dest"
  xattr -dr com.apple.quarantine "$dest/Kordon.app" 2>/dev/null || true
  open "$dest/Kordon.app"
  say "Готово: Kordon в $dest и уже в строке меню (зелёная точка рядом с часами)."
}

linux() {
  bin="$HOME/.local/bin"
  if [ "$ACTION" = "uninstall" ]; then
    stop_running
    rm -f "$bin/kordon" "$HOME/.config/autostart/kordon.desktop"
    say "Kordon удалён. Настройки остались в ~/.config/kordon."
    return
  fi
  case "$(uname -m)" in
    x86_64 | amd64) arch=amd64 ;;
    aarch64 | arm64) arch=arm64 ;;
    *) echo "Неподдерживаемая архитектура: $(uname -m)" >&2; exit 1 ;;
  esac
  mkdir -p "$bin"
  say "Скачиваю Kordon…"
  stop_running
  curl -fsSL "$BASE/kordon-linux-$arch.tar.gz" | tar -xz -C "$bin" kordon
  chmod +x "$bin/kordon"
  "$bin/kordon" autostart on
  nohup "$bin/kordon" >/dev/null 2>&1 &
  say "Готово: $bin/kordon запущен и стартует при входе."
}

case "$(uname -s)" in
  Darwin) mac ;;
  Linux) linux ;;
  *) echo "Для Windows (PowerShell): irm https://raw.githubusercontent.com/$REPO/main/scripts/install.ps1 | iex" >&2; exit 1 ;;
esac
