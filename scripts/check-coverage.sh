#!/usr/bin/env bash
set -euo pipefail

profile="${1:-coverage.out}"
if [[ ! -f "$profile" ]]; then
  echo "coverage profile not found: $profile" >&2
  exit 1
fi

report="$(go tool cover -func="$profile")"
total="$(awk '/^total:/ {gsub(/%/, "", $3); print $3}' <<<"$report")"
floor="82.0"
echo "Total coverage: ${total}% (floor ${floor}%)"
awk -v got="$total" -v min="$floor" 'BEGIN { if (got + 0 < min + 0) exit 1 }' || {
  echo "coverage ${total}% is below the ${floor}% floor" >&2
  exit 1
}

check_function() {
  local file="$1"
  local function="$2"
  local minimum="$3"
  local value
  value="$(awk -v file="$file" -v fn="$function" '
    index($1, "/" file ":") > 0 && $2 == fn {
      gsub(/%/, "", $3)
      print $3
      exit
    }
  ' <<<"$report")"
  if [[ -z "$value" ]]; then
    echo "critical coverage target not found: ${file}:${function}" >&2
    exit 1
  fi
  echo "${file}:${function} coverage: ${value}% (floor ${minimum}%)"
  awk -v got="$value" -v min="$minimum" 'BEGIN { if (got + 0 < min + 0) exit 1 }' || {
    echo "${file}:${function} coverage ${value}% is below ${minimum}%" >&2
    exit 1
  }
}

# These are the high-risk composition paths added or reused by Milestone 13.
# Function-level gates prevent aggregate coverage from hiding an untested retry
# or SSRF branch behind well-covered parser/builder code.
check_function "middleware.go" "Retry" "90.0"
check_function "observability.go" "Observe" "90.0"
check_function "retry.go" "setupReplay" "90.0"
check_function "retry.go" "retryLoop" "85.0"
check_function "security_ssrf_transport.go" "RoundTrip" "100.0"
check_function "security_ssrf_transport.go" "withSSRFDialPinning" "85.0"
