# syntax=docker/dockerfile:1.7
ARG GO_VERSION=1.27.4
FROM golang:${GO_VERSION}-bookworm

ENV DEBIAN_FRONTEND=noninteractive GOFLAGS=-buildvcs=false

RUN apt-get update \
    && apt-get install -y --no-install-recommends git make gcc libc6-dev ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -g 1000 leoxybuild \
    && useradd -m -u 1000 -g 1000 -s /bin/bash leoxybuild

WORKDIR /workspace

COPY --chmod=755 docker/build/entrypoint.sh /usr/local/bin/entrypoint.sh

USER leoxybuild

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
