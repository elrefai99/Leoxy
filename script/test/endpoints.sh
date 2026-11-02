#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh"

endpoints=("$EP1" "$EP2" "$EP3")

for i in "${!endpoints[@]}"; do
     timed "${AUTH[@]}" "$BASE_URL${endpoints[$i]}" > "$TMP_DIR/$i" &
done
wait

for i in "${!endpoints[@]}"; do
     read -r code seconds < "$TMP_DIR/$i"
     if [ "$code" -ge 200 ] && [ "$code" -lt 400 ]; then
          pass "${endpoints[$i]} -> $code (${seconds}s)"
     else
          fail "${endpoints[$i]} -> $code (${seconds}s)"
     fi
done

exit "$FAILED"
