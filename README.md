# Leoxy Proxy Service

The Leoxy proxy service exposes a small HTTP reverse proxy that forwards requests to configured upstream servers. It also provides a health-check endpoint.

## Run locally

From this directory, provide the required environment configuration and run:

```bash
go run ./cmd
```

To build the service:

```bash
go build ./...
```

To run the tests:

```bash
go test ./...
```

## Endpoints

### Health check

```http
GET /ping
```

Returns:

```text
ping
```

### Proxy routes

Configured upstreams are available under numbered paths:

```text
/proxy/1/*  -> PROXY_SERVER_1
/proxy/2/*  -> PROXY_SERVER_2
/proxy/3/*  -> PROXY_SERVER_3
/proxy/4/*  -> PROXY_SERVER_4
```

The numbered prefix is removed before forwarding the request. For example:

```http
GET /proxy/1/api/tasks
```

is forwarded to the first upstream as:

```http
GET /api/tasks
```

Routes are registered only for upstream servers with valid configured URLs.

## Project structure

```text
cmd/main.go                         Service entrypoint
internal/config/dotenv.go           Environment configuration loader
internal/server/proxyHandler.go    Reverse-proxy handler
internal/server/serverCheck.go     Health-check handler
```
