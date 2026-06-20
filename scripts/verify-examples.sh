#!/usr/bin/env bash
#
# verify-examples.sh — compile-check every book2 example against the LOCAL library.
#
# book2/ is a separate Go module that pins the library via `replace => ../`, so
# building it exercises the real, in-tree gocurl. That makes this the canonical
# guard that a public-API change (a removed/renamed export, a changed signature)
# does not silently break the documented examples — exactly the regression that
# orphaned the options builder when `Execute` was removed.
#
# It compile-checks only (no `go run`): compilation is what proves the examples
# still match the API. Running them would make live network calls (httpbin.org,
# api.github.com, …) and is intentionally out of scope here.
#
# Default is build-only — that is the deterministic API-compatibility guard. `go vet`
# is opt-in via --vet: the example sources carry intentional trailing-newline spacing
# (fmt.Println("…\n")) that vet flags but that the book prose relies on, so it is not
# part of the default pass.
#
# Not wired into CI yet (by design). It is written to BE CI-ready: non-interactive,
# no required args, color only on a TTY, and a clean non-zero exit on any failure.
#
# Usage:
#   scripts/verify-examples.sh            # build-check all book2 examples (default)
#   scripts/verify-examples.sh --vet      # also run `go vet ./...`

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
BOOK_DIR="$ROOT_DIR/book2"

RUN_VET=false
for arg in "$@"; do
    case "$arg" in
        --vet) RUN_VET=true ;;
        -h|--help)
            sed -n '2,24p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
            exit 0
            ;;
        *)
            echo "verify-examples.sh: unknown argument: $arg" >&2
            echo "try: scripts/verify-examples.sh --help" >&2
            exit 2
            ;;
    esac
done

# Color only when attached to a terminal, so CI logs stay clean.
if [ -t 1 ]; then
    RED=$'\033[0;31m'; GREEN=$'\033[0;32m'; BOLD=$'\033[1m'; NC=$'\033[0m'
else
    RED=''; GREEN=''; BOLD=''; NC=''
fi

fail() { echo "${RED}${BOLD}FAIL${NC} $*" >&2; exit 1; }

if ! command -v go >/dev/null 2>&1; then
    fail "Go toolchain not found on PATH."
fi
if [ ! -f "$BOOK_DIR/go.mod" ]; then
    fail "book2 module not found at $BOOK_DIR (expected $BOOK_DIR/go.mod)."
fi

example_count=$(find "$BOOK_DIR" -type f -name 'main.go' | wc -l | tr -d ' ')

echo "${BOLD}Verifying book2 examples against the local library${NC}"
echo "  module:   $BOOK_DIR  (replace github.com/maniartech/gocurl => ../)"
echo "  examples: $example_count main.go files"
echo "  steps:    go build ./...$([ "$RUN_VET" = true ] && echo ' + go vet ./...')"
echo

cd "$BOOK_DIR"

echo "→ go build ./..."
if ! go build ./...; then
    fail "one or more book2 examples did not compile against the current API."
fi

if [ "$RUN_VET" = true ]; then
    echo "→ go vet ./..."
    if ! go vet ./...; then
        fail "go vet reported issues in the book2 examples."
    fi
fi

echo
echo "${GREEN}${BOLD}OK${NC} all $example_count book2 examples compile against the local library."
