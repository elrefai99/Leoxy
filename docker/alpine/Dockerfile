ARG GO_VERSION=1.27.3-alpine

FROM golang:${GO_VERSION} AS builder

WORKDIR /app/app

COPY app/go.mod app/go.sum ./
RUN go mod download

WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/leoxy ./app/cli

FROM golang:${GO_VERSION}

ENV LEOXY_HOST=0.0.0.0

WORKDIR /app

COPY --from=builder /out/leoxy /usr/local/bin/leoxy
COPY --from=builder --chown=leoxy:leoxy /app/docs/html /app/docs/html

USER leoxy

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:4444/health/live || exit 1

ENTRYPOINT ["/usr/local/bin/leoxy"]
