# Leoxy

Leoxy is a lightweight HTTP reverse proxy built in Go. It routes incoming requests to configurable upstream servers based on path prefixes, with built-in health-check endpoints and request logging.

## Features

- **Reverse proxy** – forwards requests to multiple backend services using `httputil.ReverseProxy`.
- **Path-based routing** – each upstream is mapped to a configurable path prefix; the prefix is stripped before forwarding.
- **Health checks** – exposes liveness and ping endpoints for monitoring.
- **Request logging** – logs method, path, remote address, and latency for every request.
- **CLI** – start, stop, and run the service as a background process with `cobra`-based subcommands.
- **YAML configuration** – upstreams and server settings are defined in `leoxy/config.yaml` via `viper`.
- **Timeouts** – explicit read/write/idle timeouts and max header size for safety.

## Configuration

Create `leoxy/config.yaml`:

```yaml
server:
  port: "8081"

upstream:
  - name: "Backend"
    path: /api
    ip: true
    server_url: "http://localhost:7000"

  - name: "Frontend"
    path: /
    server_url: "http://localhost:3000"
```

| Key | Description |
|-----|-------------|
| `server.port` | Listen address (defaults to `:8080`; a bare number gets `:` prepended). |
| `upstream[].name` | Identifier for the upstream. |
| `upstream[].path` | Route prefix that triggers this upstream (defaults to `/`). |
| `upstream[].server_url` | Absolute URL of the backend; validated at startup. |
| `upstream[].ip` | Flag to capture and forward the client IP. |

Routes are registered only for upstreams with valid, parseable URLs.

## CLI usage

```bash
leoxy run        # start the server inline
leoxy start      # start in the background (writes .leoxy.pid)
leoxy stop       # stop the background process
leoxy version    # print version
```

## Endpoints

### Ping

```http
GET /ping
```

Returns `PONG`.

### Liveness

```http
GET /health/live
```

Returns `{"status":"ok"}`.

### Proxy

Each configured upstream is available at its `path` prefix:

```text
/<prefix>/*  ->  upstream server_url
```

The prefix is stripped before forwarding. For example, with the config above, `GET /api/tasks` is forwarded to `http://localhost:7000/tasks`.
