#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh"

body() {
     curl -s --max-time 15 "${AUTH[@]}" "$BASE_URL$1" || true
}

assert_eq "/ping status" "$(status "${AUTH[@]}" "$BASE_URL/ping")" 200
assert_eq "/ping body" "$(body /ping)" PONG

assert_eq "/health/live status" "$(status "${AUTH[@]}" "$BASE_URL/health/live")" 200
live="$(body /health/live)"
case "$live" in
     *'"status":"ok"'*) pass "/health/live body" ;;
     *) fail "/health/live body ($live)" ;;
esac

assert_eq "/health/ready status" "$(status "${AUTH[@]}" "$BASE_URL/health/ready")" "${READY_EXPECT:-200}"

if body /metrics | grep -q leoxy_requests_total; then
     pass "/metrics exposes leoxy_requests_total"
else
     fail "/metrics exposes leoxy_requests_total"
fi

if [ -n "${TCP_ADDR:-}" ]; then
     host="${TCP_ADDR%:*}"
     port="${TCP_ADDR##*:}"
     if timeout 3 bash -c "exec 3<>/dev/tcp/$host/$port" 2>/dev/null; then
          pass "tcp $TCP_ADDR accepts connections"
     else
          fail "tcp $TCP_ADDR accepts connections"
     fi
     if [ "${TCP_ECHO:-0}" = 1 ] && command -v nc >/dev/null 2>&1; then
          reply="$(printf 'ping' | timeout 5 nc -w 2 "$host" "$port" 2>/dev/null)"
          assert_eq "tcp echo through proxy" "$reply" ping
     fi
fi

if [ -n "${UDP_ADDR:-}" ]; then
     if command -v nc >/dev/null 2>&1; then
          host="${UDP_ADDR%:*}"
          port="${UDP_ADDR##*:}"
          reply="$(printf 'ping' | timeout 5 nc -u -w 2 "$host" "$port" 2>/dev/null)"
          assert_eq "udp echo through proxy" "$reply" ping
     else
          echo "SKIP  udp (nc not installed)"
     fi
fi

exit "$FAILED"
