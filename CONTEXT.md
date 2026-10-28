# Repo Context: metahub-admin (branch 58-readd-pull-through-cache)

## What changed (2026-09-15)

1. **`cmd/parse.go`** — dropped the bogus `--registry` persistent flag. It was
   a copy-paste leftover bound to the `emptyGrah` (empty-graph) variable with
   the empty-graph usage text: `--registry all` would have wiped the graph.
   No script or config consumed a registry flag → removed outright.
2. **`cmd/parse_models.go`** — `parse models` now honors the CE-declared
   per-profile config:
   - `effectiveConfigBytes(model, profile)` renders `Model.Config.Default`
     merged with profile-pinned `Config.Overrides` (Profile == profile label)
     into JSON. Rule-based overrides and `default: auto` are NOT implemented
     (warned, fall back) — pull-path/future work.
   - The rendered config is written to a temp file and passed via
     `rpConfigFile`, so `defaultConfigBytes()` in repackage pushes the
     CE-declared config instead of its built-in default.
   - `pModelsCmd` was split into `loadModels` / `stageTempConfig` /
     `repackageModelQuant` after Sonar S3776 flagged cognitive complexity 28.
3. Tests: unit tests mirroring the CE minicpm5 declaration (default,
   frame override, empty, "auto" unsupported).

## Key files

- `cmd/parse.go` — flag removal
- `cmd/parse_models.go` — config-override honoring + refactor
- `cmd/parse_models_test.go` — new tests
- `cmd/repackage.go` — shared repackage core (`repackageOnce`, `rp*` globals,
  `defaultConfigBytes()` reads `rpConfigFile` — unchanged but central)

## Architecture notes

- The pull path (`metahub-api resolveModelCentricRuntime` → metalib
  `ModelDetailGet`) reads compositions derived from repackage-created config
  artifacts (annotations profile/runtime/quant). CE overrides must therefore
  be resolved at INGEST time — that is what `parse models` now does.
- `parse models` expects quant sources cached in metahub at
  `models/<name>/<quant>` (repo path; `OrasArtifactGetByName` matches
  `c.repo` exactly, tag irrelevant). The external `quants[].source` auto-cache
  is a follow-up; the pipelines pre-stage via curl/oras.

## Open questions / deferred

- Rule-based config/latest overrides (`rule: "hardware.gpu.vram_gb >= 100"`)
  are not applied at ingest; only profile-pinned ones.
- `config.default: auto` (derive from runtime) yields a warning and falls
  back to the repackage default.
- community-edition `MetaHub-Registry/openspec` (`changes/model-centric-pull/tasks.md`) is stale —
  artifacts exist but boxes are unchecked.

## Gotchas

- `magazine.NewMagazine(cmd, MH_TABLES)` needs the full minio+neo4j config;
  pipelines pass a rendered config file.
- Sonar gate fails on any new S3776 — keep parse-style commands factored into
  small helpers.

## 2026-10-23 — open-source-metahub

- Pinned `metahub-core` at `v0.3.11-0.20261023073118-725938dd0b1e` to keep the
  module graph aligned with `metahub-api`; the admin does not call the new
  natural-key lookups.
- Added the public CI fast lane + aggregator
  (`openspec/changes/add-public-ci`).
- Use `GOWORK=off` to build against the pinned module.

## 2026-10-27 — Dependabot alerts enabled, backlog decomposed

Alerting had been off for the whole open-source session. Enabling it returned
58 open alerts here — 10 critical, 14 high, 30 medium, 4 low. The
dependency fingerprint matches `metahub-api` (both consume `metahub-core`),
minus the `go-jose` path.

Two changes now sit in `openspec/changes/`:
- `dependabot-security-alert-remediation` — outstanding work, same order as the
  API: core first, re-pin, then local leftovers.
- `refresh-go-dependencies-and-toolchain` — **already implemented**; the durable
  record of the `go-toolchain-policy` capability.

## 2026-10-27 — Remediation: core re-pin + root-module bumps

The admin now consumes core
`v0.3.12` (the `dependabot-backlog` core commit).
The re-pin closed the `x/crypto` (→ 0.58.0), `otel/sdk` (→ 1.47.0) and `grpc`
(→ 1.83.2) families and forced the module floor to **Go 1.26.0** (CI and the
Dockerfile already use 1.26). The remaining bumps were applied directly:

| module | was | now | advisory |
| --- | --- | --- | --- |
| `containerd/containerd` | 1.7.12 | **1.7.33** | GHSA-pwhc-rpq9-4c8w, GHSA-xhf5-7wjv-pqxp |
| `containers/image/v5` | 5.23.0 | **5.29.3** | GHSA-6wvf-f2vw-3425 |
| `docker/distribution` | 2.8.1+incompatible | **2.8.3+incompatible** | GHSA-hqxw-f8mx-cpmw |
| `docker/docker` | 25.0.2+incompatible | **25.0.6+incompatible** | GHSA-v23v-6jw2-98fq (critical) |
| `klauspost/compress` | 1.16.4 | **1.18.7** | GHSA-259r-337f-4rfw |

### Accepted advisories with no upstream fix (revisit 2027-01-27)

`docker/docker` is pulled in through `metahub-core`'s `magazine`/`minio`
packages and their testcontainers helpers. Both remaining highs are
**daemon-side**; this binary is a Docker *client* and never runs a daemon:

| advisory | severity | affected | why it is not reachable |
| --- | --- | --- | --- |
| GHSA-rg2x-37c3-w2rh | high | `docker/docker <= 28.5.2` | race in the daemon's `docker cp` bind-mount setup; no daemon code path is served |
| GHSA-x86f-5xw2-fm2r | high | `docker/docker <= 28.5.2` | the daemon resolves decompression binaries on `PUT /containers/{id}/archive`; that endpoint is Docker Engine's, not this service's |

There is no `distribution/distribution` alert in this repository (not in its
module graph), so no acceptance is needed for it here. The Docker Engine on
dev/CI hosts that run testcontainers is the mitigation surface for the two
daemon advisories.

Verification on the branch: `GOWORK=off go build ./...` and `go vet ./...`
clean; the short suite passes. `./test.sh` and the GoCD dev pipeline results
are recorded in the session DevDiary. Alert closure is only observable on
`main` after merge — the Dependabot alerts API reports the default branch.
