#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh"

if [ -z "$API_KEY" ]; then
     skip "security (set API_KEY and configure api_keys on Leoxy)"
fi

endpoints=("$EP1" "$EP2" "$EP3")

probe() {
     local endpoint="$1"
     shift
     status "$@" "$BASE_URL$endpoint"
}

for i in "${!endpoints[@]}"; do
     endpoint="${endpoints[$i]}"
     (
          none="$(probe "$endpoint")"
          wrong="$(probe "$endpoint" -H "X-API-Key: invalid")"
          header="$(probe "$endpoint" -H "X-API-Key: $API_KEY")"
          bearer="$(probe "$endpoint" -H "Authorization: Bearer $API_KEY")"
          echo "$none $wrong $header $bearer" > "$TMP_DIR/$i"
     ) &
done
wait

for i in "${!endpoints[@]}"; do
     read -r none wrong header bearer < "$TMP_DIR/$i"
     label="${endpoints[$i]} none=$none wrong=$wrong header=$header bearer=$bearer"
     if [ "$none" = 401 ] && [ "$wrong" = 401 ] \
          && [ "$header" -ge 200 ] && [ "$header" -lt 400 ] \
          && [ "$bearer" -ge 200 ] && [ "$bearer" -lt 400 ]; then
          pass "$label"
     else
          fail "$label"
     fi
done

if [ -n "${DENIED_METHOD:-}" ]; then
     code="$(status -X "$DENIED_METHOD" "${AUTH[@]}" "$BASE_URL$EP1")"
     assert_eq "$DENIED_METHOD is rejected by allowed_methods" "$code" 405
fi

exit "$FAILED"
