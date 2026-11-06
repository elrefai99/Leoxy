
# syntax=docker/dockerfile:1
FROM golang:1.27-bookworm

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        git \
        curl \
        make \
        gcc \
        libc6-dev \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -g 1001 leoxybuild \
    && useradd -m -u 1001 -g 1001 -s /bin/bash leoxybuild

WORKDIR /workspace

COPY app/go.mod app/go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY --chmod=755 docker/lx-2/build/entrypoint.sh ./entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
