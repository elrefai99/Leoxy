# Docker deployment

Leoxy has a multi-stage production Dockerfile at `docker/lx-2/stable/Dockerfile`. It builds the CLI binary, then runs it in Alpine as the unprivileged `leoxy` user. The container exposes port 80 and has a liveness health check against `/health/live`.

## Build and run the image

From the repository root:

```sh
docker build -f docker/lx-2/stable/Dockerfile -t leoxy:local .
docker run --rm \
  -p 80:80 \
  -v "${PWD}/leoxy:/app/leoxy" \
  leoxy:local
```

Leoxy reads `/app/leoxy/leoxy_config.yaml` in this setup. Put the configuration file in the host `leoxy/` directory before starting, or let Leoxy create its default configuration there. The image's `PORT` is 80. To publish on a different host port, change the left side of the port mapping, for example `-p 8080:80`.

## Compose deployment

The production Compose definition is `docker/lx-2/stable/docker-compose.yml`. It publishes `${LEOXY_PORT:-80}:80`, mounts `./leoxy` at `/app/leoxy`, and mounts a named volume at `/app/log`. It enables restart-on-failure recovery, a read-only root filesystem, a temporary filesystem for `/tmp`, drops Linux capabilities, and sets `no-new-privileges`.

To build and start with Compose, use a Compose file whose `build.dockerfile` points to `docker/lx-2/stable/Dockerfile` and whose context is the repository root. The checked-in Compose file currently names `docker/lx-2/Dockerfile`, which is not present in the repository; correct that path in your local deployment definition or build and run the image with the commands above.

Useful Compose operations once the file path is corrected:

```sh
docker compose -f docker/lx-2/stable/docker-compose.yml up -d --build
docker compose -f docker/lx-2/stable/docker-compose.yml ps
docker compose -f docker/lx-2/stable/docker-compose.yml logs -f leoxy
docker compose -f docker/lx-2/stable/docker-compose.yml down
```

Set `LEOXY_PORT` to change the host-side port. The container continues listening on 80. The Compose health check tests `/health/live`; it confirms the HTTP process responds, not that every upstream is healthy. Use `/health/ready` when you need the upstream readiness result.

## Docker networking and upstream services

When Leoxy runs inside a container, `localhost` and `127.0.0.1` refer to the Leoxy container's own network namespace.

The correct upstream address depends on where the application is running.

### Option 1: Upstream running on the Docker host

If your backend runs directly on the host machine, use `host.docker.internal`.

Example Leoxy configuration:

```yaml
upstream:
  - name: "Backend"
    protocol: "http"
    path: "/api"
    server_url: "http://host.docker.internal:9000"
```

Docker Desktop supports `host.docker.internal` on Windows and macOS.

For Linux Docker Engine, add the host gateway mapping:

```yaml
services:
  leoxy:
    image: mohamed0699/leoxy:latest
    ports:
      - "80:80"
    extra_hosts:
      - "host.docker.internal:host-gateway"
    volumes:
      - ./leoxy:/app/leoxy
```

The host application must listen on a reachable interface such as `0.0.0.0`, and the host firewall must allow the connection.

### Option 2: Upstream running in another container

When Leoxy and the backend run through Docker Compose, use the backend service name as the upstream hostname.

Example Compose configuration:

```yaml
services:
  leoxy:
    image: mohamed0699/leoxy:latest
    ports:
      - "80:80"
    volumes:
      - ./leoxy:/app/leoxy
    depends_on:
      - backend
    restart: unless-stopped

  backend:
    build: ./backend
    expose:
      - "9000"
    restart: unless-stopped
```

Leoxy configuration:

```yaml
upstream:
  - name: "Backend"
    protocol: "http"
    path: "/api"
    server_url: "http://backend:9000"
```

Docker Compose automatically provides service discovery between containers on the same network.

The backend does not need a published host port for Leoxy to reach it. The `expose` declaration is optional and documents the internal application port.

### Troubleshooting

If an upstream connection fails:

1. Verify that the backend is running and listening on the expected port.
2. Verify that the backend listens on a reachable interface.
3. Check that both containers share a Docker network.
4. Use the Compose service name instead of `localhost` for container-to-container communication.
5. Use `host.docker.internal` for host-based applications.
6. Check firewall rules and container logs.

Remember that `depends_on` controls startup order but does not guarantee that the backend is ready to accept connections.

## Configuration, persistence, and secrets

Mount the directory containing `leoxy_config.yaml` at `/app/leoxy` so configuration survives container replacement. The current watcher and background CLI use `leoxy/log` relative to the container working directory (`/app`), so their default output is under `/app/leoxy/log`; the Compose volume at `/app/log` does not cover that default path. Mount or redirect the actual data directory if reports and logs need to persist. With `docker run`, a host mount at `/app/leoxy` also persists the default logs and watcher reports alongside the config.

Keep credentials out of images and source control. Supply secrets through the container environment and ensure the mounted configuration has only the settings needed for the deployment. When using TLS certificate paths, mount the certificate files read-only and configure paths that exist inside the container.

## Available Dockerfiles

| Dockerfile | Intended use | Notes |
| --- | --- | --- |
| `docker/lx-2/stable/Dockerfile` | Production Linux image used by the Docker workflow | Multi-stage build; runs as the `leoxy` user and includes `wget` plus a `/health/live` health check. |
| `docker/lx-2/alpine/Dockerfile` | Alpine-based Leoxy image | Builds a Linux binary in Alpine. It currently declares `USER leoxy` without creating that user in the final stage, and its health check calls `wget` although the final stage does not install it. Resolve those image requirements before using this target. |
| `docker/lx-2/alpine/windows/Dockerfile.windows` | Windows container image | Builds on Windows Server Core and runs on Nano Server LTSC 2022. Build it with a Windows container engine and compatible Windows host. |
| `docker/lx-2/build/Dockerfile` | Development/CI image | Its entrypoint verifies Go modules, checks formatting, runs `go vet`, runs race-enabled tests, and builds the CLI. |
| `docker/benchmark/Dockerfile` | Benchmark client image | This file still refers to the old `tools/` module layout. The current benchmark source is `test/benchmark`; use that Go module directly until the Dockerfile is updated. |

Build the Alpine Linux target from the repository root with:

```sh
docker build -f docker/lx-2/alpine/Dockerfile -t leoxy:alpine .
```

For Windows containers, run the build on a Windows container host:

```powershell
docker build -f docker/lx-2/alpine/windows/Dockerfile.windows -t leoxy:windows .
```

## Docker workflow

`.github/workflows/docker.yml` is a reusable workflow (`workflow_call`). It builds the stable image, validates and builds the stable Compose service, then pushes versioned and `latest` Docker Hub tags only when `version.json` changed and Docker Hub credentials are available. Otherwise, it completes a local image build without pushing. The workflow currently targets the stable Dockerfile; it does not build the Alpine, Windows, build-helper, or benchmark images.

The Compose validation/build steps use `docker/lx-2/stable/docker-compose.yml`. Its `build.dockerfile` currently points to `docker/lx-2/Dockerfile`, which is absent; update that path to `docker/lx-2/stable/Dockerfile` for those workflow steps to succeed.

