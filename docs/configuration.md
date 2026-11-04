# Leoxy configuration

This guide explains the sample configuration in [`config/leoxy_config.yaml`](../config/leoxy_config.yaml). At runtime, Leoxy reads `leoxy/leoxy_config.yaml` relative to its working directory. Copy the sample there and edit it for your service. If the runtime file is missing, Leoxy creates a small default configuration; it does not automatically copy this repository sample.

You can choose another file with the CLI's `--config` option. `--data-dir` changes the working directory used for relative paths and runtime data.

## Basic shape

The YAML has three main sections:

```yaml
server:    # Listener, timeouts, and shared security policies
watcher:   # Optional process resource monitoring
upstream:  # HTTP, TCP, and UDP destinations
```

Keep indentation consistent. `upstream` is a YAML list; each list entry defines one route or stream listener. The sample includes a backend, a frontend, and TCP/UDP examples. Remove or adapt entries you do not use.

## Server

```yaml
server:
  host: "127.0.0.1"
  port: "4444"
  timeout:
    connect: 5
    read: 30
    write: 30
    idle: 60
    request: 60
```

`host` and `port` define the HTTP listener. The default host in the sample is loopback, so remote clients cannot connect unless you set a reachable interface such as `0.0.0.0`. `PORT` overrides `server.port`, and `LEOXY_HOST` overrides `server.host`.

Timeout values are in seconds. `connect`, `read`, `write`, and `idle` are applied to the HTTP server and proxy transport. `request` has a configuration default but is not currently applied by the runtime server code.

### Shared security settings

`server.security` applies policies shared by HTTP traffic:

| Key | Meaning |
| --- | --- |
| `rate_limit`, `rate_limit_burst` | Legacy per-client limit inherited by non-root upstreams that do not set a positive per-client limit. |
| `global_rate_limit`, `global_rate_burst` | Shared HTTP-wide token bucket, applied before route dispatch. |
| `route_rate_limit`, `route_rate_burst` | Shared per-route limit defaults passed to route middleware. Per-upstream route settings are normally used for route policy. |
| `api_keys` | Accepted API key values. Clients send `X-API-Key` or a bearer credential. Prefer environment variables for real secrets. |
| `jwt_secret`, `jwt_issuer`, `jwt_audience` | JWT verification settings. Tokens use HS256 and must include an expiration. A configured secret must be at least 32 bytes. |
| `require_mtls` | Require a client certificate for TLS connections. Needs server certificate/key and `client_ca`. |
| `allow_cidrs`, `deny_cidrs` | Network ranges allowed or denied by the access policy. Values use CIDR notation. |
| `allowed_methods` | HTTP methods accepted by the access policy. An empty list does not impose a method allowlist. |
| `allowed_paths` | Optional path patterns accepted by the access policy. An empty list does not impose a path allowlist. |
| `max_body` | Maximum HTTP request body size in MiB. |
| `redis_addr`, `redis_password`, `redis_db` | Redis connection settings for the shared global rate limiter. |
| `redis_fail_open` | If true, use the local global limiter when Redis is unavailable. |
| `trusted_proxy_cidrs` | Proxy ranges trusted when resolving the original client address from forwarding headers. |
| `internal_cidrs` | Networks allowed to access internal endpoints such as `/health/ready` and `/leoxy`. |
| `tls_cert`, `tls_key` | Set both to enable HTTPS on the HTTP listener. |
| `client_ca` | CA bundle used to verify client certificates when mTLS is enabled. |

Rate values are tokens per minute with a burst capacity. Zero disables the corresponding limit. The shared global limit is charged before route policies; use per-upstream `security` for route-specific limits.

The runtime environment overrides supported by the current loader include `RATE_LIMIT`, `JWT_SECRET`, `JWT_ISSUER`, `JWT_AUDIENCE`, `API_KEYS`, and `REDIS_PASSWORD`. `API_KEYS` is a comma-separated list. Keep secrets out of committed YAML files.

## HTTP upstreams

An HTTP entry uses `path` to select a route and one or more upstream URLs:

```yaml
upstream:
  - name: "Backend"
    protocol: "http"
    path: "/api"
    servers:
      - "http://localhost:60459"
      - "http://localhost:60460"
```

Leoxy forwards requests whose path starts with `/api` to this upstream. Multiple entries in `servers` are load balanced with failover. For one target, use `server_url` instead of `servers`. Use `path: "/"` for a catch-all route and make sure no other HTTP entry uses the same path. Leoxy reserves its own endpoint paths, including `/ping`, `/health`, and `/metrics`.

The sample also contains `ip`, `listen`, `limit_request`, and `body`. `listen` is for stream protocols, not HTTP routes. `body` supplies the route body limit when `security.max_body` is unset or non-positive. `ip` controls the proxy's client IP forwarding behavior. `limit_request` is present in the configuration schema but is not currently enforced by the runtime.

The validator accepts HTTP-family protocol labels (`http`, `https`, `http2`, `graphql`, and `websocket`, as well as empty for compatibility). Routing uses the same HTTP reverse proxy handler for these labels; the protocol label does not create a separate listener.

### Per-upstream security

Each HTTP upstream can contain its own `security` block. This is where to set route-specific `rate_limit` / `rate_limit_burst`, `route_rate_limit` / `route_rate_burst`, authentication, CIDR rules, allowed methods/paths, and request body size. Upstream security values are scoped to that route. The global HTTP rate limit still applies across all routes.

For non-root paths such as `/api`, if the upstream's per-IP `rate_limit` is zero or missing, its per-IP limit inherits `server.security.rate_limit` and its burst. A positive upstream rate takes precedence. The root route does not inherit that legacy per-IP limit. Route body size uses `security.max_body` if positive, otherwise the upstream `body` value.

## TCP and UDP upstreams

Stream entries use a listen address on Leoxy and a target address on the backend:

```yaml
- name: "GameTCP"
  protocol: "tcp"
  listen: ":7001"
  server_url: "127.0.0.1:7000"

- name: "TelemetryUDP"
  protocol: "udp"
  listen: ":7002"
  server_url: "127.0.0.1:7003"
```

These listeners are independent of the HTTP port. Their security block supports `allow_cidrs`, `deny_cidrs`, and `max_conns_per_ip`. HTTP authentication, methods, paths, and request body settings do not apply to TCP/UDP traffic. Use a protocol plus a non-empty `listen` and `server_url`; listener addresses for the same protocol must be unique.

## Watcher

The `watcher` block configures Go process resource monitoring. `status: "on"` enables periodic samples; `memory_limit_mb` must be positive or the monitor remains disabled. `interval_seconds` controls the sampling interval. On reaching the limit, Leoxy writes a report and, when `heap_profile` is true, a Go heap profile. It can also POST the JSON report to `webhook_url`. `cooldown_seconds` prevents repeated reports too frequently. See [Watcher guide](watcher/watcher.md) for fields, output, and metric details.

Environment overrides supported by the config loader are `MEMORY_LIMIT_MB`, `PROJECT_NAME`, and `Watcher_WEBHOOK_URL` (spelling is as implemented). Watcher files are written under `leoxy/log` relative to the runtime working directory by default.

## Validate and troubleshoot

Leoxy validates the loaded config before opening listeners. Common startup errors include unsupported protocol labels, repeated HTTP paths, reserved HTTP paths, invalid CIDR ranges, negative limits, invalid TLS combinations, and malformed JWT settings.

Check the following when a route does not work:

1. Confirm the process is using the file you edited (`--config` can select a different file).
2. Confirm HTTP upstream values use absolute URLs such as `http://localhost:7000`.
3. Confirm each HTTP path is unique and begins with `/`.
4. Confirm TCP/UDP `listen` and target addresses have the correct host-and-port form.
5. Check `/health/live` for process liveness and `/health/ready` for HTTP upstream readiness.

For container paths and volumes, see [Docker deployment](docker.md). For commands and CLI path flags, see [CLI usage](cli.md).

