#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh"

if [ -z "${LB_HEADER:-}" ]; then
     skip "lb (set LB_HEADER to the header your upstreams use to identify themselves)"
fi

N="${N:-60}"
LB_EP="${LB_EP:-$EP1}"
MIN_BACKENDS="${MIN_BACKENDS:-2}"
out="$TMP_DIR/lb"

for _ in $(seq "$N"); do
     curl -s -o /dev/null -D - -w 'CODE:%{http_code}\n' --max-time 15 "${AUTH[@]}" "$BASE_URL$LB_EP" || true
done | tr -d '\r' > "$out"

counts="$(grep -i "^$LB_HEADER:" "$out" | sed 's/^[^:]*: *//' | sort | uniq -c)"
backends="$(printf '%s\n' "$counts" | grep -c .)"

echo "$LB_EP distribution:"
printf '%s\n' "$counts" | sed 's/^/   /'

if [ "$backends" -ge "$MIN_BACKENDS" ]; then
     pass "served by $backends backends (min $MIN_BACKENDS)"
     min="$(printf '%s\n' "$counts" | awk '{print $1}' | sort -n | head -n 1)"
     floor=$(( N / (backends * 2) ))
     if [ "$min" -ge "$floor" ]; then
          pass "round-robin is balanced (least used backend got $min, floor $floor)"
     else
          fail "round-robin is unbalanced (least used backend got $min, floor $floor)"
     fi
else
     fail "served by $backends backends (min $MIN_BACKENDS)"
fi

if [ "${EXPECT_NO_5XX:-1}" = 1 ]; then
     errors="$(grep -c '^CODE:5' "$out")"
     assert_eq "no 5xx while balancing" "$errors" 0
fi

exit "$FAILED"
