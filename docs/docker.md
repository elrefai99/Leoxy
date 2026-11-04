# Docker deployment

Leoxy has a multi-stage production Dockerfile at `docker/lx-2/stable/Dockerfile`. It builds the CLI binary, then runs it in Alpine as the unprivileged `leoxy` user. The container exposes port 4444 and has a liveness health check against `/health/live`.

## Build and run the image

From the repository root:

```sh
docker build -f docker/lx-2/stable/Dockerfile -t leoxy:local .
docker run --rm \
  -p 4444:4444 \
  -v "${PWD}/leoxy:/app/leoxy" \
  leoxy:local
```

Leoxy reads `/app/leoxy/leoxy_config.yaml` in this setup. Put the configuration file in the host `leoxy/` directory before starting, or let Leoxy create its default configuration there. The image's `PORT` is 4444. To publish on a different host port, change the left side of the port mapping, for example `-p 8080:4444`.

## Compose deployment

The production Compose definition is `docker/lx-2/stable/docker-compose.yml`. It publishes `${LEOXY_PORT:-4444}:4444`, mounts `./leoxy` at `/app/leoxy`, and mounts a named volume at `/app/log`. It enables restart-on-failure recovery, a read-only root filesystem, a temporary filesystem for `/tmp`, drops Linux capabilities, and sets `no-new-privileges`.

To build and start with Compose, use a Compose file whose `build.dockerfile` points to `docker/lx-2/stable/Dockerfile` and whose context is the repository root. The checked-in Compose file currently names `docker/lx-2/Dockerfile`, which is not present in the repository; correct that path in your local deployment definition or build and run the image with the commands above.

Useful Compose operations once the file path is corrected:

```sh
docker compose -f docker/lx-2/stable/docker-compose.yml up -d --build
docker compose -f docker/lx-2/stable/docker-compose.yml ps
docker compose -f docker/lx-2/stable/docker-compose.yml logs -f leoxy
docker compose -f docker/lx-2/stable/docker-compose.yml down
```

Set `LEOXY_PORT` to change the host-side port. The container continues listening on 4444. The Compose health check tests `/health/live`; it confirms the HTTP process responds, not that every upstream is healthy. Use `/health/ready` when you need the upstream readiness result.

## Configuration, persistence, and secrets

Mount the directory containing `leoxy_config.yaml` at `/app/leoxy` so configuration survives container replacement. The current watcher and background CLI use `leoxy/log` relative to the container working directory (`/app`), so their default output is under `/app/leoxy/log`; the Compose volume at `/app/log` does not cover that default path. Mount or redirect the actual data directory if reports and logs need to persist. With `docker run`, a host mount at `/app/leoxy` also persists the default logs and watcher reports alongside the config.

Keep credentials out of images and source control. Supply secrets through the container environment and ensure the mounted configuration has only the settings needed for the deployment. When using TLS certificate paths, mount the certificate files read-only and configure paths that exist inside the container.

## Build and CI helper files

`docker/lx-2/build/Dockerfile` and its `entrypoint.sh` form a development/CI image. Its entrypoint verifies Go modules, checks formatting, runs `go vet`, runs race-enabled tests, and builds the CLI. `docker/benchmark/Dockerfile` is an older benchmark image definition and references a `tools/` layout that is not the current benchmark module path; use `test/benchmark` directly unless that Dockerfile is updated for the current repository layout.

