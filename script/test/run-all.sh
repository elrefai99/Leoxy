#!/usr/bin/env bash
dir="$(dirname "$0")"
failed=0

for name in endpoints load ratelimit auth; do
     echo "== $name =="
     bash "$dir/$name.sh" || failed=1
done

exit "$failed"
