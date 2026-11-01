#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh"

endpoints=("$EP1" "$EP2" "$EP3")

for i in "${!endpoints[@]}"; do
     hit "${endpoints[$i]}" > "$TMP_DIR/$i" &
done
wait

for i in "${!endpoints[@]}"; do
     read -r code time < "$TMP_DIR/$i"
     if [ "$code" -ge 200 ] && [ "$code" -le "$EXPECT_MAX" ]; then
          check "${endpoints[$i]} -> $code (${time}s)" pass
     else
          check "${endpoints[$i]} -> $code (${time}s)" fail
     fi
done

exit "$FAILED"
