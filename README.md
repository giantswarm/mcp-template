[![CircleCI](https://dl.circleci.com/status-badge/img/gh/giantswarm/{MCP-NAME}/tree/main.svg?style=svg)](https://dl.circleci.com/status-badge/redirect/gh/giantswarm/{MCP-NAME}/tree/main)

# {MCP-NAME}

SHORT_DESC_PLACEHOLDER

This repo was bootstrapped from
[`giantswarm/mcp-template`](https://github.com/giantswarm/mcp-template) — a
template for Go MCP servers built on
[`mark3labs/mcp-go`](https://github.com/mark3labs/mcp-go) and
[`giantswarm/mcp-oauth`](https://github.com/giantswarm/mcp-oauth).

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
`team-PLACEHOLDER` / `SHORT_DESC_PLACEHOLDER` placeholder, deletes itself
plus the bootstrap-gate workflow, and runs `go mod tidy`. Commit the
result.

## Quickstart

```bash
make run        # stdio transport (Claude Desktop)
make run-http   # streamable-HTTP, no OAuth (curl / mcp-inspector)
make test       # go test ./...
make helm-lint  # validate the chart
```

For the full OAuth flow, deploy the chart against a cluster with Dex
configured — see `docs/ARCHITECTURE.md`.

## What's inside

| Path                | Purpose                                                       |
| ------------------- | ------------------------------------------------------------- |
| `main.go` + `cmd/`  | cobra entry — `serve.go` wires everything top-to-bottom       |
| `internal/server/`  | logging, OTEL, health, OAuth wiring, transport selection      |
| `internal/tools/`   | example tools (`things_list`, `things_get`, `things_create`)  |
| `internal/example/` | placeholder domain client + fake — replace with your upstream |
| `helm/{MCP-NAME}/`  | Helm chart (ServiceMonitor, NetworkPolicy, hardened SC)       |
| `docs/`             | architecture                                                  |

## Configuration

Every knob is an env var; flags override. Highlights:

| Variable               | Default          | Purpose                            |
| ---------------------- | ---------------- | ---------------------------------- |
| `MCP_TRANSPORT`        | streamable-http  | stdio \| sse \| streamable-http    |
| `MCP_ADDR`             | :8080            | MCP HTTP listener                  |
| `METRICS_ADDR`         | :9091            | /metrics, /healthz, /readyz        |
| `OAUTH_ENABLED`        | false            | Set true in production             |
| `OAUTH_ISSUER`         | —                | This server's own /oauth/* base    |
| `DEX_ISSUER_URL`       | —                | Upstream Dex / OIDC issuer         |
| `OAUTH_STORAGE`        | memory           | memory \| valkey                   |
| `VALKEY_ADDR`          | —                | required when storage=valkey       |

## Releasing

CHANGELOG follows [Keep a Changelog](https://keepachangelog.com). Tag
`vX.Y.Z`; the CircleCI architect orb pushes the multi-arch image to
gsoci and the chart to giantswarm-catalog.

## Security

Report vulnerabilities per `SECURITY.md`. The image is distroless,
runs as non-root with `readOnlyRootFilesystem: true`, drops all Linux
capabilities, and the chart's NetworkPolicy default-denies egress
except DNS + the allowlist.
