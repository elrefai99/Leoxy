#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh"

LIMIT_MB="${BODY_LIMIT_MB:-10}"
EP="${BODY_EP:-$EP1}"
limit=$(( LIMIT_MB * 1024 * 1024 ))

post_known() {
     head -c "$1" /dev/zero | curl -s -o /dev/null -w '%{http_code}' --max-time 60 \
          -X POST -H 'Content-Type: application/octet-stream' -H 'Expect: 100-continue' \
          "${AUTH[@]}" --data-binary @- "$BASE_URL$EP" || true
}

post_chunked() {
     head -c "$1" /dev/zero | curl -s -o /dev/null -w '%{http_code}' --max-time 60 \
          -X POST -H 'Content-Type: application/octet-stream' \
          "${AUTH[@]}" -T - "$BASE_URL$EP" || true
}

assert_accepted "known length, 1 KB under limit" "$(post_known $(( limit - 1024 )))"
assert_accepted "known length, exactly at limit" "$(post_known "$limit")"
assert_eq "known length, 1 byte over limit" "$(post_known $(( limit + 1 )))" 413
assert_eq "chunked, 1 byte over limit" "$(post_chunked $(( limit + 1 )))" 413

exit "$FAILED"
