#!/usr/bin/env bash
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
WORK="$(mktemp -d)"
KEY="e2e-key"
PIDS=()

cleanup() {
     for pid in "${PIDS[@]}"; do
          kill "$pid" 2>/dev/null || true
     done
     rm -rf "$WORK"
}
trap cleanup EXIT

cd "$ROOT" || exit 1

go build -o "$WORK/leoxy-bin" ./app/cli || exit 1
mkdir -p "$WORK/leoxy"
cp script/test/config.e2e.yaml "$WORK/leoxy/config.yaml"

python3 script/test/upstream.py &
PIDS+=("$!")

( cd "$WORK" && exec env API_KEYS="$KEY" ./leoxy-bin run ) &
PIDS+=("$!")

ready=0
for _ in $(seq 40); do
     if curl -s -o /dev/null -H "X-API-Key: $KEY" http://127.0.0.1:4444/ping; then
          ready=1
          break
     fi
     sleep 0.5
done

if [ "$ready" -ne 1 ]; then
     echo "Leoxy did not become ready"
     exit 1
fi

API_KEY="$KEY" \
EP1=/api/test EP2=/ EP3=/health/ready \
LB_HEADER=X-Served-By \
BODY_EP=/body BODY_LIMIT_MB=1 \
RL_EPS="/limited-a /limited-b /limited-c" \
DENIED_METHOD=TRACE \
TCP_ADDR=127.0.0.1:7001 TCP_ECHO=1 \
UDP_ADDR=127.0.0.1:7002 \
N=100 C=20 \
bash script/test/run-all.sh
