#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh"

BURST="${BURST:-150}"
read -ra eps <<< "${RL_EPS:-$EP1 $EP2 $EP3}"

for i in "${!eps[@]}"; do
     seq "$BURST" | xargs -P 20 -I{} curl -s -o /dev/null -w '%{http_code}\n' --max-time 15 \
          "${AUTH[@]}" "$BASE_URL${eps[$i]}" > "$TMP_DIR/$i" &
done
wait

for i in "${!eps[@]}"; do
     echo "${eps[$i]}"
     sort "$TMP_DIR/$i" | uniq -c | sed 's/^/   /'
     if grep -q '^429' "$TMP_DIR/$i"; then
          pass "${eps[$i]} returned 429 under burst"
     else
          fail "${eps[$i]} returned 429 under burst"
     fi
done

found=""
for _ in $(seq 10); do
     headers="$(curl -s -o /dev/null -D - --max-time 15 "${AUTH[@]}" "$BASE_URL${eps[0]}" | tr -d '\r')"
     if printf '%s\n' "$headers" | grep -q '^HTTP/[0-9.]* 429'; then
          found="$headers"
          break
     fi
done

if printf '%s\n' "$found" | grep -qi '^Retry-After: 60'; then
     pass "429 carries Retry-After: 60"
else
     fail "429 carries Retry-After: 60"
fi

exit "$FAILED"
