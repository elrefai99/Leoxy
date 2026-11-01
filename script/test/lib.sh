#!/usr/bin/env bash
set -uo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:4444}"
EP1="${EP1:-/api/test}"
EP2="${EP2:-/}"
EP3="${EP3:-/health/ready}"
API_KEY="${API_KEY:-}"
EXPECT_MAX="${EXPECT_MAX:-399}"
FAILED=0
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

HDR=()
if [ -n "$API_KEY" ]; then
     HDR=(-H "X-API-Key: $API_KEY")
fi

hit() {
     curl -s -o /dev/null -w '%{http_code} %{time_total}' --max-time 15 "${HDR[@]}" "$BASE_URL$1" || true
}

check() {
     if [ "$2" = "pass" ]; then
          echo "PASS  $1"
     else
          echo "FAIL  $1"
          FAILED=1
     fi
}
