# Docker Images

This directory contains Dockerfiles for the Leoxy Linux, Windows, and build-helper images.

## Alpine Linux

Build from the repository root:

```sh
docker build -f docker/alpine/Dockerfile -t leoxy:alpine .
```

The image listens on port `4444`. Run it with your Leoxy configuration mounted:

```sh
docker run --rm -p 4444:4444 -v "${PWD}/leoxy:/app/leoxy" leoxy:alpine
```

## Windows

Build on a Windows host using a compatible Windows container engine:

```powershell
docker build -f docker/alpine/windows/Dockerfile.windows -t leoxy:windows .
```

The image is based on Nano Server LTSC 2022 and listens on port `4444`.

## Stable image

Build the stable Linux image from the repository root:

```sh
docker build -f docker/stable/Dockerfile -t leoxy:stable .
```

## Build helper

`build/Dockerfile` provides a Go build environment with its entrypoint script. It is intended for development and CI.

## Configuration

Leoxy reads configuration from `leoxy/leoxy_config.yaml`. Mount the host `leoxy` directory at `/app/leoxy` to provide and persist that configuration. See the [configuration guide](../docs/configuration.md) for details.
