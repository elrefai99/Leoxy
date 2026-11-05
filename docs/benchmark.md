# Benchmarking Leoxy

The standalone benchmark client in `test/benchmark` sends HTTP requests to a running Leoxy endpoint. It reports response status counts, accepted and rejected requests, request throughput, response throughput, latency percentiles, and memory allocation statistics for the benchmark process. It does not measure Leoxy's memory directly.

## Build and run

Run from the repository root:

```sh
go run ./test/benchmark/cmd \
  -url http://localhost:4444/api \
  -duration 30s \
  -concurrency 100
```

The benchmark module can also be built separately:

```sh
cd test/benchmark
go build -o bench ./cmd
./bench -url http://localhost:4444/api -duration 30s -concurrency 100
```

`-url`, `-duration`, and `-concurrency` are required. The test ends when its duration expires or its optional request limit is reached.

## Options

| Flag | Default | Description |
| --- | --- | --- |
| `-url` | empty | Target HTTP URL |
| `-method` | `GET` | HTTP method |
| `-duration` | zero | Run duration, such as `30s` or `2m` |
| `-concurrency` | zero | Number of concurrent workers |
| `-request` | `0` | Maximum total request attempts; zero means no request-count limit |
| `-body` | empty | Literal request body |
| `-body-size` | `0` | Generate a request body containing this many `a` bytes; overrides `-body` when positive |
| `-chunked` | `false` | Send a non-empty body without a known content length |
| `-header` | none | Add a request header as `Key: Value`; repeat for multiple headers. `Host` sets the HTTP host field |
| `-expect` | 2xx and 3xx | Comma-separated status codes treated as accepted, for example `200,204` |
| `-pprof` | empty | Optional URL to fetch a Leoxy heap profile after the load run |
| `-profile-out` | `docs/leoxy-heap.pb.gz` | File path for the downloaded profile |

Example with a POST request, authentication header, fixed request count, and profile download:

```sh
go run ./test/benchmark/cmd \
  -url http://localhost:4444/api/items \
  -method POST \
  -body '{"name":"sample"}' \
  -header 'Content-Type: application/json' \
  -header 'X-API-Key: replace-with-your-key' \
  -duration 1m \
  -request 50000 \
  -concurrency 50 \
  -pprof http://localhost:4444/debug/pprof/heap \
  -profile-out ./heap.pb.gz
```

The profiling endpoint is disabled unless the Leoxy process has `LEOXY_PPROF=true`. Treat that endpoint as privileged diagnostic access and keep it off public interfaces.

## Reading results

Latency is the time for a request and its response body to be read. `p50`, `p95`, and `p99` are percentiles of completed requests; `max` is the slowest. Accepted statuses default to 200–399, and failed transport/read attempts are reported as `error`. A nonzero benchmark exit status indicates at least one rejected request, or a profile download failure.

The memory section measures the client process: heap in use, bytes allocated during the run, and allocation count. It is not the proxy's memory usage. For the proxy heap profile, use `-pprof`; analyze the resulting file with:

```sh
go tool pprof -top ./heap.pb.gz
```

For meaningful comparisons, keep the target route, request body, expected statuses, runtime environment, and load settings consistent. Avoid saturating the benchmark machine; running the client on a separate host helps distinguish client limits from proxy performance.

