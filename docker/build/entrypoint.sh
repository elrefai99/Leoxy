#!/usr/bin/env bash
set -Eeuo pipefail

export GOWORK=off

for module in app test/benchmark; do
    echo "==> $module"
    pushd "$module" >/dev/null
    go mod verify
    if [ -n "$(gofmt -l .)" ]; then
        gofmt -l .
        echo "Formatting check failed"
        exit 1
    fi
    go vet ./...
    CGO_ENABLED=1 go test -race ./...
    popd >/dev/null
done

echo "==> Building Leoxy"
mkdir -p dist

(cd app && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../dist/leoxy ./cli)

echo "==> CI completed successfully"
