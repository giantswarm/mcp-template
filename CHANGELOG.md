# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `pkg/project` carries the build identifiers the generated Makefile and the architect `go-build` job stamp at link time (`version`, `gitSHA`, `buildTimestamp`); `main.go` injects `project.Version()` into the root command, so `version`, `--version`, the OTEL service version and the MCP server info report the release version instead of `dev`.
- `self-update` installs the latest GitHub release only after its cosign Sigstore bundle verifies for a CircleCI build of the repo (`github.com/giantswarm/selfupdate-cosign`); a release without a bundle or a download that does not match its signature is refused and the installed binary stays as it is. Needs the `cli` flavour in the repo's giantswarm/github team file (documented in the README); `scripts/init.sh` derives the GitHub slug from `--module`.
- Running the binary without a subcommand starts the server, same as `serve`; the root command carries a Long description and `SilenceUsage`, and `cmd/doc.go`, `cmd/root_test.go`, `cmd/version_test.go` and `cmd/selfupdate_test.go` cover the command surface.
- Chart unit tests (`helm/{MCP-NAME}/tests/deployment_test.yaml`, `make helm-test`), `.helmignore`, `.golangci.yml` goconst tuning, `.gitignore` entries for the CI scratch files that flip the VCS stamp to `+dirty` (`.ldflags`, `.platforms`, `bin-dist/`), `make test-vet` / `make govulncheck`, and the `architecture.mdc` / `dev_workflow.mdc` cursor rules — the developer surface of mcp-kubernetes and mcp-timescale.
- The bootstrap gate builds the generated repo's binary, runs its tests and `version`, checks that a `dev` build refuses `self-update`, and runs the chart unit tests.
- Adopt `github.com/giantswarm/mcp-toolkit` v0.1.0 for cross-cutting plumbing. `cmd/serve.go` now imports `logging.New`, `tracing.Init`, `health.New`, `httpx.Run`, `responsecap.New`, and `timeout.New` instead of carrying inline copies. Per-tool middleware (`responsecap` with default 128 KiB cap, `timeout` with 30s default) is wired from day one so every MCP scaffolded from this template inherits both protections.

### Changed

- `Makefile.custom.mk` no longer overrides the generated `build`, `test`, `lint`, `fmt`, `vet` and `run` targets — the generated `build` is what stamps `pkg/project`; `make run` (stdio) is now `make run-stdio`.
- Bump `github.com/giantswarm/mcp-oauth` to v1.0.0.
- Bump `github.com/mark3labs/mcp-go` v0.49.0 → v0.52.0 to align with the rest of the Giant Swarm Go MCP fleet.
- `Config` no longer carries a `LogFormat` field. Format is auto-selected by `mcp-toolkit/logging` (JSON when `KUBERNETES_SERVICE_HOST` is set, text otherwise). The `LOG_FORMAT` env-var override is dropped — override at the call site in `cmd/serve.go` if a specific MCP needs a fixed format.

### Removed

- `cmd.version`, `cmd.commit` and `cmd.date` — nothing stamped them, so every released binary printed `dev`.
- `internal/server/{health,logging,tracing}.go` — superseded by `mcp-toolkit/{health,logging,tracing}`.
- `Auth.IssuerHealthURL()` and the matching `oauth-issuer` readiness probe. The toolkit's `health` package follows the principle that `/readyz` should not probe shared downstreams: a transient Dex hiccup would otherwise flip every replica's `/readyz` simultaneously and the Service yanks its last endpoint. Token validation failures continue to surface to individual callers as 401/503.
