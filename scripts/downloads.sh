#!/bin/bash
# How many times Kordon was downloaded, per version and per file, plus the
# repository's visitors over the last 14 days. Needs the GitHub CLI (gh).
#
#   scripts/downloads.sh
set -euo pipefail
repo=ef1m03/kordon

echo "Скачивания по версиям"
gh api "repos/$repo/releases" --paginate -q '.[] | [.tag_name, ([.assets[] | select(.name | test("^(Kordon|kordon)-")) | .download_count] | add // 0)] | @tsv' |
  awk -F'\t' '{ printf "  %-8s %6d\n", $1, $2; total += $2 } END { printf "  %-8s %6d\n", "всего", total }'

echo
echo "По системам (все версии)"
gh api "repos/$repo/releases" --paginate -q '.[].assets[] | select(.name | test("^(Kordon|kordon)-")) | [.name, .download_count] | @tsv' |
  awk -F'\t' '
    { n = $1
      if (n ~ /macos/) k = "macOS"
      else if (n ~ /windows-amd64/) k = "Windows x64"
      else if (n ~ /windows-arm64/) k = "Windows ARM"
      else if (n ~ /linux/) k = "Linux"
      else k = n
      s[k] += $2 }
    END { for (k in s) printf "  %-12s %6d\n", k, s[k] }' | sort

echo
gh api "repos/$repo" -q '"Звёзды: \(.stargazers_count)   форки: \(.forks_count)   наблюдают: \(.subscribers_count)"'
if views=$(gh api "repos/$repo/traffic/views" -q '"Просмотры страницы за 14 дней: \(.count), уникальных: \(.uniques)"' 2>/dev/null); then
  echo "$views"
  gh api "repos/$repo/traffic/popular/referrers" -q '.[] | "  откуда: \(.referrer) - \(.uniques)"' 2>/dev/null | head -8
fi
