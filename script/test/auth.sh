#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh"

endpoints=("$EP1" "$EP2" "$EP3")

probe() {
     curl -s -o /dev/null -w '%{http_code}' --max-time 15 "${@:2}" "$BASE_URL$1"
}

for i in "${!endpoints[@]}"; do
     (
          none=$(probe "${endpoints[$i]}")
          wrong=$(probe "${endpoints[$i]}" -H "X-API-Key: invalid")
          valid=$(probe "${endpoints[$i]}" -H "X-API-Key: $API_KEY")
          echo "$none $wrong $valid" > "$TMP_DIR/$i"
     ) &
done
wait

for i in "${!endpoints[@]}"; do
     read -r none wrong valid < "$TMP_DIR/$i"
     if [ "$none" = 401 ] && [ "$wrong" = 401 ] && [ "$valid" -lt 400 ]; then
          check "${endpoints[$i]} none=$none wrong=$wrong valid=$valid" pass
     else
          check "${endpoints[$i]} none=$none wrong=$wrong valid=$valid" fail
     fi
done

exit "$FAILED"
