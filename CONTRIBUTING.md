# Contributing to Leoxy

## Before you start

- Bugs and features: use the issue forms. Blank issues are disabled.
- Vulnerabilities: do not open an issue. See [SECURITY.md](SECURITY.md).
- For large changes, open a feature request first and agree on scope before coding.

## Repository layout

| Path | Purpose |
| --- | --- |
| `app/cli` | Cobra CLI: run, start, restart, stop, version |
| `app/internal/core` | Server bootstrap, routing, TLS, shutdown, platform process helpers |
| `app/internal/http_server` | Proxy, load balancer, health, TCP, UDP, shared transport |
| `app/internal/security` | Access control, JWT, rate limits (memory and Redis), body limit, internal-only |
| `app/internal/file` | Config loading, validation, defaults |
| `app/internal/common` | Client IP, request logger, metrics |
| `tools` | Benchmark tool, separate Go module |
| `docker/lx-2` | Dockerfile and Compose file |
| `script` | End-to-end and load test scripts, build and service files |
| `example` | Sample `config.yaml` |
| `docs` | CLI docs, security notes, static page |

`go.work` joins `app` and `tools`. Keep new implementation code under `app/internal/` unless it is an entrypoint.

## Requirements

- Go version from `.go-version`
- GNU Make (optional)
- Docker, only for image changes

## Development

```sh
make run
make build
```

A default `leoxy/config.yaml` is created on first start in the working directory. See [docs/cli.md](docs/cli.md) and `example/config.yaml`.

## Before opening a PR

From the repository root:

```sh
gofmt -l app tools
go vet ./app/... ./tools/...
go test ./app/... ./tools/...
go build ./app/... ./tools/...
```

`gofmt -l` must print nothing. CI runs vet, test and build per module, plus `go test -race` and `govulncheck` in the security workflow. Run the race detector locally for concurrency changes:

```sh
go test -race ./app/... ./tools/...
```

For OS-specific code (process, signal, detach files):

```sh
GOOS=windows go build ./app/cli
GOOS=linux GOARCH=arm64 go build ./app/cli
```

For image changes:

```sh
docker compose -f docker/lx-2/docker-compose.yml config --quiet
docker compose -f docker/lx-2/docker-compose.yml build
```

For end-to-end behavior, `make test-e2e` uses `script/test/` (needs `python3` and `curl`). For performance-sensitive changes, compare before and after with the benchmark tool in [tools/README.md](tools/README.md).

## Code guidelines

- Standard Go style, `gofmt` clean.
- Keep changes small and scoped to the request.
- Add `_test.go` files beside the package, table-driven where several inputs apply. Behavior changes need regression coverage.
- Never hold a mutex while calling the next handler or doing network I/O.
- Do not trust client forwarding headers except through `trusted_proxy_cidrs`.
- Do not log secrets, tokens, API keys, request bodies, or query strings.
- No secrets, tokens, or private upstream URLs in code, tests, examples, or logs.
- New config fields: add the `mapstructure` tag, validate in `file.Validate`, and document in `README.md`, `docs/security.md` when relevant, and `example/config.yaml`.
- Prefer the standard library. Justify any new dependency.

## Security-sensitive changes

Changes to authentication, rate limiting, client IP detection, TLS, TCP/UDP listeners, or default config must state the threat addressed, include a test that fails without the change, and note any new default or breaking behavior.

## Commits and pull requests

Title format: `type(scope): imperative summary`. Lowercase, no trailing period, under about 70 characters.

| Type | Use for | Version bump |
| --- | --- | --- |
| `feat` | New capability, any file count | minor |
| `fix` | Bug live in production | patch |
| `refactor` | 3 or more files, no new capability | patch |
| `chore` | Small updates, max 2 files | patch |
| `docs` | Documentation only, no scope | none |

Scope is the package or feature name used in `git log` (`security`, `proxy`, `tcp`, `docker`). Omit it for repo-wide changes. Use `!` after the scope for a breaking change; it needs maintainer agreement first.

Each PR must:
1. Bump the version in `version.json` and the `version` constant in `app/cli/main.go`, and keep `app/README.md` in step. `release.yml` tags `v<version>` from `version.json` on every push to `main`, so skipping the bump means no release.
2. Add an entry at the top of `CHANGELOG.md` under Added, Changed, or Fixed. Docs-only PRs go under `[Unreleased]` with no bump.
3. Use a body with exactly three summary bullets, the version change, and the exact verification commands and results, including what you did not test.
4. Keep the three bullets identical in the PR body and the changelog.

```text
## Summary
- <change 1>
- <change 2>
- <change 3>

## Version
<old> -> <new>, see CHANGELOG.md

## Verification
<commands, results, and what was not verified>
```

Do not write "safe" or "backward compatible" without evidence in the verification section.

## CI

Every push and PR runs vet, test and build for `app` and `tools`, golangci-lint, and the security workflow. On `main`, the Docker workflow builds the image, validates Compose, and pushes to Docker Hub when credentials exist and `version.json` changed. A push to `main` also creates a GitHub release if the tag does not exist.

## AI-assisted contributions

Agents must follow [AGENTS.md](AGENTS.md): ask before changing code, keep changes minimal, never read or modify `.env` files, and never run destructive git or shell commands. Disclose AI assistance in one line at the end of the PR body.

## License

By contributing, you agree your contributions are licensed under the Apache License 2.0.
