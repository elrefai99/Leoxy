#!/usr/bin/env bash
dir="$(dirname "$0")"
failed=0

for name in endpoints network security lb body load ratelimit; do
     echo "== $name =="
     bash "$dir/$name.sh" || failed=1
done

exit "$failed"
