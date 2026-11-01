#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh"

BURST="${BURST:-150}"
endpoints=("$EP1" "$EP2" "$EP3")

for i in "${!endpoints[@]}"; do
     seq "$BURST" | xargs -P 30 -I{} curl -s -o /dev/null -w '%{http_code}\n' --max-time 15 "${HDR[@]}" "$BASE_URL${endpoints[$i]}" > "$TMP_DIR/$i" &
done
wait

for i in "${!endpoints[@]}"; do
     if grep -q '^429' "$TMP_DIR/$i"; then
          check "${endpoints[$i]} returned 429 under burst" pass
     else
          check "${endpoints[$i]} returned 429 under burst" fail
     fi
done

exit "$FAILED"
