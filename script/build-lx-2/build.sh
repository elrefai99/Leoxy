#!/usr/bin/env sh
set -eu

build() {
  goos="$1"
  goarch="$2"
  extension=""

  if [ "$goos" = windows ]; then
    extension=".exe"
  fi

  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -ldflags="-s -w" -o "dist/leoxy-$goos-$goarch$extension" ./app/cli
}

build_linux() {
  build linux amd64
  build linux arm64
}

build_windows() {
  build windows amd64
  build windows arm64
}

mkdir -p dist

case "${1:-all}" in
  linux) build_linux ;;
  windows) build_windows ;;
  all)
    build_linux
    build_windows
    ;;
  *)
    echo "usage: $0 [linux|windows|all]" >&2
    exit 2
    ;;
esac
