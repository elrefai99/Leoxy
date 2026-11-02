#!/usr/bin/env bash
set -o pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:4444}"
EP1="${EP1:-/api/test}"
EP2="${EP2:-/}"
EP3="${EP3:-/health/ready}"
API_KEY="${API_KEY:-}"
FAILED=0
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

AUTH=()
if [ -n "$API_KEY" ]; then
     AUTH=(-H "X-API-Key: $API_KEY")
fi

status() {
     curl -s -o /dev/null -w '%{http_code}' --max-time 15 "$@" || true
}

timed() {
     curl -s -o /dev/null -w '%{http_code} %{time_total}' --max-time 15 "$@" || true
}

pass() {
     echo "PASS  $1"
}

fail() {
     echo "FAIL  $1"
     FAILED=1
}

skip() {
     echo "SKIP  $1"
     exit 0
}

assert_eq() {
     if [ "$2" = "$3" ]; then
          pass "$1 ($2)"
     else
          fail "$1 (got $2, want $3)"
     fi
}

assert_accepted() {
     if [ "$2" != "413" ] && [ "$2" != "000" ]; then
          pass "$1 ($2)"
     else
          fail "$1 (got $2, want anything but 413 or 000)"
     fi
}
