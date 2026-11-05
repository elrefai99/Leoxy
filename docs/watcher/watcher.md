# Resource watcher

The watcher is part of the Leoxy process. When enabled, it samples Go runtime memory, heap, goroutines, and cumulative CPU time at a configured interval. If its memory threshold is reached, it writes a JSON report, can save a heap profile, and can send the report to an HTTP webhook. It does not restart Leoxy or enforce a memory cap.

## Enable and configure

Set `watcher.status` to `"on"` in `leoxy/leoxy_config.yaml`:

```yaml
watcher:
  status: "on"
  project_name: "my-service"
  memory_limit_mb: 512
  interval_seconds: 5
  cooldown_seconds: 300
  heap_profile: true
  webhook_url: "https://monitor.example.com/leoxy"
```

`memory_limit_mb` must be positive for sampling to start. `interval_seconds` controls how often the process checks usage; a non-positive value falls back to five seconds. `cooldown_seconds` limits repeated reports after an over-limit sample; a non-positive value falls back to five minutes. `project_name` labels reports and defaults to the current directory name when empty. `heap_profile` controls whether each threshold report includes a heap profile. `webhook_url` is optional and must be an HTTP or HTTPS URL with a host.

The sample field named `memory_bytes` represents Go runtime total memory classes minus released heap memory; it is not an operating-system/container RSS measurement. `heap_bytes` reports heap object bytes. The threshold is checked against `memory_bytes`.

## Reports and webhook

The watcher writes into `leoxy/log` by default, relative to Leoxy's working directory. A threshold breach creates `memory-report-<UTC timestamp>.json`, with project and host labels, process ID, time, uptime, configured limit, sampled memory/heap/goroutine/CPU values, and the log directory size. If enabled and successful, the JSON report also contains the path of a `heap-<UTC timestamp>.pprof` file.

The webhook receives the report as a JSON `POST` with `Content-Type: application/json`. The request has a five-second timeout. HTTP responses with status 300 or greater and network errors are logged. Reports and profiles are pruned separately to retain at most the 20 newest matching files.

## Metrics and shutdown

Each watcher sample updates the memory metric exposed at `/metrics`. The watcher stops its sampling goroutine during normal process shutdown. It logs when the monitor starts, when a threshold is crossed, and when report/profile/webhook operations fail.

The repository's example [`leoxy_config.yaml`](../config/leoxy_config.yaml) shows watcher keys. Its webhook URL and memory limit are examples; replace them for your environment. In the production container, the default `leoxy/log` resolves to `/app/leoxy/log`; persist the mounted `/app/leoxy` directory or otherwise redirect the runtime working/data directory to keep reports across container replacement.

