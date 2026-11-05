
#!/usr/bin/env bash

set -Eeuo pipefail

echo "==> Verifying Go modules"
go mod verify

echo "==> Checking formatting"
if [ -n "$(gofmt -l .)" ]; then
    gofmt -l .
    echo "Formatting check failed"
    exit 1
fi

echo "==> Running go vet"
go vet ./...

echo "==> Running tests with race detection"
go test -race ./...

echo "==> Building Leoxy"
mkdir -p dist
CGO_ENABLED=1 go build -trimpath -o dist/leoxy ./app/cli

echo "==> CI completed successfully"
