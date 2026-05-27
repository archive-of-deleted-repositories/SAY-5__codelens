#!/usr/bin/env bash
set -euo pipefail
THRESHOLD="${1:-55}"
PROFILE="${2:-cover.out}"
pct=$(go tool cover -func="$PROFILE" \
  | grep -E 'internal/(graph|index|store|semantic|cache)/' \
  | awk '{gsub("%","",$NF); sum+=$NF; n++} END {if (n>0) printf "%.1f", sum/n; else print "0"}')
echo "internal coverage: ${pct}% (floor ${THRESHOLD}%)"
awk -v p="$pct" -v t="$THRESHOLD" 'BEGIN { exit (p+0 >= t+0) ? 0 : 1 }' || {
  echo "coverage ${pct}% is below floor ${THRESHOLD}%"; exit 1; }
