# syntax=docker/dockerfile:1.7
ARG GO_VERSION=1.27.4
ARG ALPINE_VERSION=3.22

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG LEOXY_VERSION=1.14.3
ENV CGO_ENABLED=0 GOWORK=off

WORKDIR /src/app

COPY app/go.mod app/go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY app/ ./

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X main.version=v${LEOXY_VERSION}" -o /out/leoxy ./cli


FROM alpine:${ALPINE_VERSION}

ARG LEOXY_VERSION=1.14.3
ARG VCS_REF=unknown
ARG BUILD_DATE=unknown

LABEL org.opencontainers.image.title="leoxy" \
      org.opencontainers.image.version="${LEOXY_VERSION}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.source="https://github.com/elrefai99/Leoxy" \
      org.opencontainers.image.licenses="Apache-2.0"

ENV LEOXY_HOST=0.0.0.0 PORT=80

RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 10001 leoxy \
    && adduser -S -D -H -u 10001 -G leoxy leoxy \
    && mkdir -p /app/leoxy /app/log /app/docs/html \
    && chown -R leoxy:leoxy /app

WORKDIR /app

COPY --from=builder /out/leoxy /usr/local/bin/leoxy
COPY --chown=leoxy:leoxy docs/html /app/docs/html

USER 10001:10001

EXPOSE 80

STOPSIGNAL SIGTERM

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null "http://127.0.0.1:${PORT:-80}/health/live" || exit 1

ENTRYPOINT ["/usr/local/bin/leoxy"]
