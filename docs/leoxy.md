# Leoxy

Leoxy is a Go reverse proxy that listens on one HTTP(S) endpoint and routes traffic to configured upstreams. It also supports separate TCP and UDP listeners. This guide describes the current `app/` implementation.

## How requests are routed

HTTP upstreams are selected by path prefix. Each upstream can use `server_url` for one target or `servers` for several targets; the proxy load balances across the configured servers and can fail over when a target is unavailable. Configure distinct paths such as `/api` and `/` for API and frontend traffic. Paths reserved for Leoxy endpoints cannot be used as upstream routes.

TCP and UDP upstreams use their own `listen` address and forward to `server_url`. They do not use HTTP paths or HTTP authentication. Their access policy supports allowed and denied CIDR ranges and per-IP connection limits.

## Run Leoxy

Build from the repository root:

```sh
go build -o leoxy ./app/cli
```

On Windows, use `go build -o leoxy.exe ./app/cli`. Run the binary from a directory where you want Leoxy's `leoxy/` data directory to live. The default configuration path is `leoxy/leoxy_config.yaml`; if it is absent, Leoxy creates a default configuration.

```sh
./leoxy
```

The CLI also supports background operation:

```sh
./leoxy start
./leoxy restart
./leoxy stop
./leoxy version
```

`start` appends output to `leoxy/log/leoxy.log` and records the process ID under `leoxy/keys/`. `--config` selects a configuration file and `--data-dir` sets the working/data directory. See [CLI usage](cli.md).

## Configuration

The configuration is YAML with `server`, `upstream`, and optional `watcher` sections. A minimal HTTP setup is:

```yaml
server:
  host: "127.0.0.1"
  port: "4444"

upstream:
  - name: "API"
    protocol: "http"
    path: "/api"
    servers:
      - "http://localhost:7000"
      - "http://localhost:7001"
```

Set `path: "/"` for a catch-all frontend route. Upstream URLs must include a scheme and host. TCP/UDP entries specify `protocol`, `listen`, and a host-and-port `server_url` instead. See the [configuration guide](configuration.md) for field descriptions, behavior, and setup instructions.

`PORT` overrides `server.port`, and `LEOXY_HOST` overrides `server.host`. `RATE_LIMIT`, `JWT_SECRET`, `JWT_ISSUER`, `JWT_AUDIENCE`, `API_KEYS`, and `REDIS_PASSWORD` override global security settings. Keep credentials in environment variables.

Security settings can be applied globally under `server.security` and per HTTP upstream under `upstream[].security`. They include per-IP and route/global rate limits, API key and HS256 JWT authentication, CIDR access rules, allowed methods and paths, body limits, trusted proxy ranges, TLS, and optional Redis-backed global rate limiting. JWT tokens require an expiration. TLS is enabled when both `tls_cert` and `tls_key` are set; mutual TLS also requires a client CA.

## Built-in endpoints

| Path | Purpose |
| --- | --- |
| `/ping` | Basic ping response |
| `/health/live` | Liveness check; does not depend on upstream availability |
| `/health/ready` | Readiness check for configured HTTP upstreams; restricted to internal CIDRs when configured |
| `/metrics` | Request/response metrics, including watcher memory samples |
| `/leoxy` | Bundled Leoxy information page, restricted to internal CIDRs |
| `/debug/pprof/` | Go profiling endpoints, enabled only with `LEOXY_PPROF=true` |

## Docker

For the container image and Compose setup, see [Docker deployment](docker.md). The container uses port 4444 by default and keeps its configuration and logs in mounted data locations.

