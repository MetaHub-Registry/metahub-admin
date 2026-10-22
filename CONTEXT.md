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
