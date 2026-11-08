ARG GO_VERSION=1.27.3
ARG LEOXY_VERSION=1.13.3

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app/app

COPY app/go.mod app/go.sum ./

RUN go mod download

WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/leoxy ./app/cli

FROM alpine:3.22

ENV LEOXY_HOST=0.0.0.0

COPY docker/stable/setup.sh /tmp/setup.sh
RUN /bin/sh /tmp/setup.sh && rm /tmp/setup.sh
    
WORKDIR /app

COPY --from=builder /out/leoxy /usr/local/bin/leoxy
COPY --chown=leoxy:leoxy docs/html /app/docs/html

USER leoxy

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:80/health/live || exit 1

ENTRYPOINT ["/usr/local/bin/leoxy"]
