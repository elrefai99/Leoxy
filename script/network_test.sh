#!/usr/bin/env bash
set -u

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RESULT="baseline"
REPORT="$ROOT/docs/$RESULT/report.txt"
FAILED=0

cd "$ROOT" || exit 1
mkdir -p "docs/$RESULT"

if ! command -v go >/dev/null 2>&1; then
     echo "ERROR: go is not installed or not in path."
     exit 1
fi

if ! command -v rg >/dev/null 2>&1; then
     echo "ERROR: rg is not installed or not in path."
     exit 1
fi

run_check() {
     local name="$1"
     shift

     {
          echo
          echo "==================================================="
          echo "Running: $name"
          echo "==================================================="
     } | tee -a "$REPORT"

     "$@" 2>&1 | tee -a "$REPORT"
     local status=${PIPESTATUS[0]}

     if [ "$status" -ne 0 ]; then
          echo "FAILED (exit code: $status)" | tee -a "$REPORT"
          FAILED=1
     else
          echo "PASSED" | tee -a "$REPORT"
     fi
}

{
     echo "Leoxy Baseline"
     echo "Date: $(date)"
     echo "Go version: $(go version)"
     echo "GOOS: $(go env GOOS) - GOARCH: $(go env GOARCH)"
     echo
     echo "Git status:"
     git status --short 2>&1 || true
} > "$REPORT"

run_check "Go test" go test ./app/... ./tools/...
run_check "Go vet" go vet ./app/... ./tools/...
run_check "Go race detector" go test -race ./app/... ./tools/...

run_check "Network listeners and servers" \
     rg -n '(ListenAndServe|net\.Listen|ListenUDP|Accept\(|ReadFromUDP)' app tools

run_check "HTTP proxy and handlers" \
     rg -n '(ReverseProxy|RoundTrip|Transport|ServeHTTP|http\.Handler)' app tools

run_check "Data transfer and connections" \
     rg -n '(io\.Copy|ReadFrom|WriteTo|DialContext|SetDeadline|Close\()' app tools

run_check "Benchmarks" \
     go test ./app/... -run '^$' -bench . -benchmem

echo
echo "==================================================="
echo "Baseline finished"
echo "Report: $REPORT"
echo "==================================================="

if [ "$FAILED" -ne 0 ]; then
     echo "Some checks failed. Review the report."
     exit 1
fi

echo "All checks passed"
