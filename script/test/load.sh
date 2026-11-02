#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh"

N="${N:-200}"
C="${C:-20}"
endpoints=("$EP1" "$EP2" "$EP3")

start="$(date +%s)"
for i in "${!endpoints[@]}"; do
     seq "$N" | xargs -P "$C" -I{} curl -s -o /dev/null -w '%{http_code}\n' --max-time 15 \
          "${AUTH[@]}" "$BASE_URL${endpoints[$i]}" > "$TMP_DIR/$i" &
done
wait
elapsed=$(( $(date +%s) - start ))

for i in "${!endpoints[@]}"; do
     echo "${endpoints[$i]}"
     sort "$TMP_DIR/$i" | uniq -c | sed 's/^/   /'
     if grep -qE '^(5|000)' "$TMP_DIR/$i"; then
          fail "${endpoints[$i]} no 5xx or connection errors"
     else
          pass "${endpoints[$i]} no 5xx or connection errors"
     fi
done

echo "total: $(( N * 3 )) requests in ${elapsed}s"
exit "$FAILED"
