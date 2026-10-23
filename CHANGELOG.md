# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- The MCP endpoint refuses a request body larger than 1 MiB before mcp-go reads it whole (CVE-2026-108859, no fixed mcp-go release yet; the scan ignores it until 2026-11-12). ([#187](https://github.com/giantswarm/mcp-template/issues/187))

### Added

- The template repository has its own CircleCI pipeline: `.circleci/config.yml` builds, tests and nancy-scans the template's Go code (`go-build-mcp-template`), with nothing released. The generated repository's pipeline is `.circleci/config.derived.yml`, which `scripts/init.sh` fills in and moves over it, and the bootstrap gate checks that it names the generated repository. The `go-build` job of both pipelines runs in the `architect` context, which the orb needs to sign the binaries.
- `pkg/project` carries the build identifiers the generated Makefile and the architect `go-build` job stamp at link time (`version`, `gitSHA`, `buildTimestamp`); `main.go` injects `project.Version()` into the root command, so `version`, `--version`, the OTEL service version and the MCP server info report the release version instead of `dev`.
- `self-update` installs the latest GitHub release only after its cosign Sigstore bundle verifies for a CircleCI build of the repo (`github.com/giantswarm/selfupdate-cosign`); a release without a bundle or a download that does not match its signature is refused and the installed binary stays as it is. Needs the `cli` flavour in the repo's giantswarm/github team file (documented in the README); `scripts/init.sh` derives the GitHub slug from `--module`.
- Running the binary without a subcommand starts the server, same as `serve`; the root command carries a Long description and `SilenceUsage`, and `cmd/doc.go`, `cmd/root_test.go`, `cmd/version_test.go` and `cmd/selfupdate_test.go` cover the command surface.
- Chart unit tests (`helm/{MCP-NAME}/tests/deployment_test.yaml`, `make helm-test`), `.helmignore`, `.golangci.yml` goconst tuning, `.gitignore` entries for the CI scratch files that flip the VCS stamp to `+dirty` (`.ldflags`, `.platforms`, `bin-dist/`), `make test-vet`, and the `architecture.mdc` / `dev_workflow.mdc` cursor rules — the developer surface of mcp-kubernetes and mcp-timescale.
- The bootstrap gate builds the generated repo's binary, runs its tests and `version`, checks that a `dev` build refuses `self-update`, and runs the chart unit tests.
- Adopt `github.com/giantswarm/mcp-toolkit` v0.1.0 for cross-cutting plumbing. `cmd/serve.go` now imports `logging.New`, `tracing.Init`, `health.New`, `httpx.Run`, `responsecap.New`, and `timeout.New` instead of carrying inline copies. Per-tool middleware (`responsecap` with default 128 KiB cap, `timeout` with 30s default) is wired from day one so every MCP scaffolded from this template inherits both protections.

### Changed

- The example tools are described with mcp-toolkit's `tooldesc.Describe` (summary line, then details), so a server generated from the template starts with the tool-description conventions ([giantswarm#38592](https://github.com/giantswarm/giantswarm/issues/38592)).
- The example tools declare all four annotations instead of reporting mcp-go's defaults: `things_list` and `things_get` are read-only, not destructive and idempotent (a `readOnly()` option); `things_create` is a write that is neither destructive nor idempotent; `openWorldHint` went from true to false on all three.
- `docs/tools.md` lists every tool as a model reads it (`make tools-doc`); `TestToolsDocIsCurrent` keeps it current, `TestToolDescriptionBudgets` holds the descriptions to `tooldesc.DefaultBudget` and `TestReadOnlyToolsAreNotDestructive` checks the read-only annotations.
- mcp-toolkit is bumped to v0.4.2.
- `Makefile.custom.mk` no longer overrides the generated `build`, `test`, `lint`, `fmt`, `vet` and `run` targets — the generated `build` is what stamps `pkg/project`; `make run` (stdio) is now `make run-stdio`.
- Bump `github.com/giantswarm/mcp-oauth` to v1.0.0.
- Bump `github.com/mark3labs/mcp-go` v0.49.0 → v0.52.0 to align with the rest of the Giant Swarm Go MCP fleet.
- `Config` no longer carries a `LogFormat` field. Format is auto-selected by `mcp-toolkit/logging` (JSON when `KUBERNETES_SERVICE_HOST` is set, text otherwise). The `LOG_FORMAT` env-var override is dropped — override at the call site in `cmd/serve.go` if a specific MCP needs a fixed format.
- Update github.com/giantswarm/mcp-oauth to v1.4.6 (giantswarm/mcp-template#150)
- Update architect to v10.11.1 (giantswarm/mcp-template#155)
- Update architect to v10.12.1 (giantswarm/mcp-template#169)
- Update github.com/prometheus/client_golang to v1.25.0 (giantswarm/mcp-template#173)
- Update github.com/mark3labs/mcp-go to v1.2.1 (giantswarm/mcp-template#175)
- Update github.com/giantswarm/go-selfupdate to v1.6.3 (giantswarm/mcp-template#183)
- Update architect to v10.14.4 (giantswarm/mcp-template#208)
- Update architect to v10.14.10 (giantswarm/mcp-template#218)
- Update github.com/giantswarm/selfupdate-cosign to v0.3.13 (giantswarm/mcp-template#228)

### Removed

- `cmd.version`, `cmd.commit` and `cmd.date` — nothing stamped them, so every released binary printed `dev`.
- `internal/server/{health,logging,tracing}.go` — superseded by `mcp-toolkit/{health,logging,tracing}`.
- `Auth.IssuerHealthURL()` and the matching `oauth-issuer` readiness probe. The toolkit's `health` package follows the principle that `/readyz` should not probe shared downstreams: a transient Dex hiccup would otherwise flip every replica's `/readyz` simultaneously and the Service yanks its last endpoint. Token validation failures continue to surface to individual callers as 401/503.

### Fixed

- `self-update` replaces the binary, symbolic links resolved, with a single rename (`selfupdatecosign.Install`, selfupdate-cosign v0.3.0). go-selfupdate's own swap moved the binary aside before it moved the new one in, so a process started in between found none, and concurrent updates could lose it. Now a process started meanwhile runs the old binary or the new one, several updates may run at once, and the binary keeps its mode.
- The chart's `helm.sh/chart` label is valid for any chart version: the 63-character cut of a long version (a branch build, or the `<tag>+<digest>` helm-controller installs) could end in `.`, `_` or `--.`, and the API server refused every labelled object; `trimAll "-._"` now trims the whole run, and `helm/{MCP-NAME}/tests/chart_label_test.yaml` asserts it.
