[![CircleCI](https://dl.circleci.com/status-badge/img/gh/giantswarm/{MCP-NAME}/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/giantswarm/{MCP-NAME}/tree/main)

# {MCP-NAME}

SHORT_DESC_PLACEHOLDER

This repo was bootstrapped from
[`giantswarm/mcp-template`](https://github.com/giantswarm/mcp-template) — a
template for Go MCP servers built on
[`mark3labs/mcp-go`](https://github.com/mark3labs/mcp-go),
[`giantswarm/mcp-oauth`](https://github.com/giantswarm/mcp-oauth), and
[`giantswarm/mcp-toolkit`](https://github.com/giantswarm/mcp-toolkit) (logger,
OTEL bootstrap, /healthz + /readyz, graceful HTTP shutdown, response-cap
middleware, per-tool timeout middleware).

## Bootstrap (template only — delete this section after running)

If you just clicked **Use this template** and cloned the result, run:

```bash
./scripts/init.sh \
  --name=mcp-foo \
  --team=team-atlas \
  --module=github.com/giantswarm/mcp-foo \
  --short-desc='What this MCP does in one line' \
  --audience=mcp-foo \
  --port=8080
```

This rewrites every `{MCP-NAME}` / `mcp-template` / `internal/example` /
`team-PLACEHOLDER` / `SHORT_DESC_PLACEHOLDER` placeholder (the `self-update`
GitHub slug follows `--module`: `github.com/giantswarm/mcp-foo` →
`giantswarm/mcp-foo`), deletes itself plus the bootstrap-gate workflow, and
runs `go mod tidy`. Commit the result.

Then register the repo in giantswarm/github (`repositories/<team>.yaml`) with
`ci.generate: true` and `gen.flavours: [generic, app, cli]` —
[mcp-timescale's entry](https://github.com/giantswarm/github/blob/main/repositories/team-bumblebee.yaml)
is the reference. `generic` + `app` give the generated CircleCI pipeline,
pre-commit and auto-release workflows; `cli` is what attaches the six
platform binaries (linux, darwin, windows on amd64 and arm64) and their cosign
Sigstore bundles to every GitHub Release. **`self-update` only works with the
`cli` flavour**: without release binaries and bundles it refuses every release
as unsigned ("no signature bundle").

## Quickstart

```bash
make run-stdio  # stdio transport (Claude Desktop)
make run-http   # streamable-HTTP, no OAuth (curl / mcp-inspector)
make test       # go test ./...
make helm-lint  # validate the chart
make helm-test  # chart unit tests (helm unittest plugin)
```

For the full OAuth flow, deploy the chart against a cluster with Dex
configured — see `docs/ARCHITECTURE.md`.

## CLI

```text
{MCP-NAME}               # same as `{MCP-NAME} serve`
{MCP-NAME} serve         # --transport stdio|sse|streamable-http, --mcp-addr, --metrics-addr, --debug
{MCP-NAME} version       # version, commit and build time
{MCP-NAME} self-update   # verified update to the latest release
```

`version` and `--version` report the release version: `pkg/project` carries
the identifiers the generated Makefile and the architect `go-build` job stamp
at link time (`make build` stamps them locally); a plain `go build` reports
Go's VCS build info (`+dirty` with untracked files) and falls back to `dev`.

`self-update` downloads the latest release's binary together with its bundle
and installs it only after the bundle verifies for a CircleCI build of the
repo against the Sigstore public-good trust root
([`giantswarm/selfupdate-cosign`](https://github.com/giantswarm/selfupdate-cosign)).
A release without a bundle, or a download that does not match its signature,
is refused and the installed binary stays as it is; a `dev` build refuses to
update itself.

## What's inside

| Path                | Purpose                                                                |
| ------------------- | ---------------------------------------------------------------------- |
| `main.go` + `cmd/`  | cobra entry — `serve.go` wires everything top-to-bottom; `version`, `self-update` |
| `pkg/project/`      | build identifiers (version, commit, build time) stamped at link time   |
| `internal/server/`  | template-specific wiring: config, OAuth, transport mux, /metrics       |
| `internal/tools/`   | example tools (`things_list`, `things_get`, `things_create`)           |
| `internal/example/` | placeholder domain client + fake — replace with your upstream          |
| `helm/{MCP-NAME}/`  | Helm chart (ServiceMonitor, NetworkPolicy, hardened SC); `tests/` are helm-unittest suites |
| `.cursor/rules/`    | architecture invariants and the development workflow for agents        |
| `docs/`             | architecture                                                           |

Cross-cutting plumbing (slog factory, OTEL init, /healthz + /readyz,
graceful HTTP shutdown, response-cap and per-tool timeout middleware) is
imported from [`giantswarm/mcp-toolkit`](https://github.com/giantswarm/mcp-toolkit)
in `cmd/serve.go`.

`make build`, `make test`, `make lint` and `make fmt` come from the generated
Makefile; `Makefile.custom.mk` adds `run-stdio`, `run-http`, `test-vet`,
`govulncheck`, `helm-lint`, `helm-template` and `helm-test`.
`pre-commit run --all-files` is what CI runs: gofmt, goimports, golangci-lint
(gosec, goconst — tuned in `.golangci.yml`), `go mod tidy`, the values schema.

## Configuration

Every knob is an env var; flags override. The OAuth knobs come straight from
[`mcp-oauth/oauthconfig`](https://pkg.go.dev/github.com/giantswarm/mcp-oauth/oauthconfig)
— refer to that package for the full surface (`OAUTH_TRUSTED_AUDIENCES`,
`OAUTH_TRUSTED_REDIRECT_SCHEMES`, `OAUTH_ALLOW_LOCALHOST_REDIRECT_URIS`, the
`*_FILE` Kubernetes-secret variants, etc.). Highlights:

| Variable                       | Default         | Purpose                                                    |
| ------------------------------ | --------------- | ---------------------------------------------------------- |
| `MCP_TRANSPORT`                | streamable-http | stdio \| sse \| streamable-http                            |
| `MCP_ADDR`                     | :8080           | MCP HTTP listener                                          |
| `METRICS_ADDR`                 | :9091           | /metrics, /healthz, /readyz                                |
| `OAUTH_ENABLED`                | false           | Set true in production                                     |
| `OAUTH_PROVIDER`               | —               | dex \| google \| github                                    |
| `OAUTH_ISSUER`                 | —               | This server's own /oauth/* base (loopback exempts http://) |
| `OAUTH_DEX_ISSUER_URL`         | —               | Upstream Dex issuer (when provider=dex)                    |
| `OAUTH_DEX_CLIENT_ID`          | —               | Dex client ID                                              |
| `OAUTH_DEX_CLIENT_SECRET[_FILE]` | —             | Dex client secret (or path to a mounted secret)            |
| `OAUTH_STORAGE_BACKEND`        | memory          | memory \| valkey                                           |
| `OAUTH_VALKEY_ADDR`            | —               | required when backend=valkey                               |
| `OAUTH_VALKEY_PASSWORD[_FILE]` | —               | optional Valkey auth                                       |
| `OAUTH_ENCRYPTION_KEY[_FILE]`  | —               | optional 32-byte AES-GCM key (base64 or hex)               |

## Releasing

CHANGELOG follows [Keep a Changelog](https://keepachangelog.com). With
`ci.generate: true` in the team file, releases are cut automatically from
Conventional Commit PR titles (`feat:`, `fix:`, `feat!:`; `chore:` and `ci:`
bump the patch too, `docs:` releases nothing): the Auto Release workflow tags
the merge commit, and the CircleCI tag pipeline pushes the multi-arch image to
gsoci, the chart to giantswarm-catalog and — with the `cli` flavour — the six
platform binaries with their cosign bundles to the GitHub Release. The CI
itself is generated by devctl and kept in sync by the align-files workflow;
change the team file, not the generated files.

## Security

Report vulnerabilities per `SECURITY.md`. The image is distroless,
runs as non-root with `readOnlyRootFilesystem: true`, drops all Linux
capabilities, and the chart's NetworkPolicy default-denies egress
except DNS + the allowlist.
