#!/usr/bin/env bash
set -euo pipefail

profile="${1:?coverage profile is required}"
floor="${2:?coverage floor is required}"
label="${3:-package}"
total="$(go tool cover -func="$profile" | awk '/^total:/ {gsub(/%/, "", $3); print $3}')"
echo "${label} coverage: ${total}% (floor ${floor}%)"
awk -v got="$total" -v min="$floor" 'BEGIN { if (got + 0 < min + 0) exit 1 }' || {
  echo "${label} coverage ${total}% is below ${floor}%" >&2
  exit 1
}
