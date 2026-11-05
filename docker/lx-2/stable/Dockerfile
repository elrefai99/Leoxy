FROM golang:1.27.3 AS builder

WORKDIR /app/app

COPY app/go.mod app/go.sum ./

RUN go mod download

WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/leoxy ./app/cli

FROM alpine:3.22

RUN apk add --no-cache ca-certificates wget \
    && addgroup -S leoxy \
    && adduser -S -G leoxy leoxy \
    && mkdir -p /app/leoxy /app/log /app/docs/html \
    && chown -R leoxy:leoxy /app

WORKDIR /app

COPY --from=builder /out/leoxy /usr/local/bin/leoxy
COPY --from=builder --chown=leoxy:leoxy /app/docs/html /app/docs/html

USER leoxy

EXPOSE 4444

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:4444/health/live || exit 1

ENTRYPOINT ["/usr/local/bin/leoxy"]
