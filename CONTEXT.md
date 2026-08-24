# Context: metahub-admin

> This file helps AI agents understand the repo quickly.

## model-centric-pull (session isc-session-v2, 2026-07-07) — branch `53-isc-session-v2`

`parse models` v1 — ingest declarative CE `models/` entries. Commit `3fe68e0`.

**Key files:**
- `cmd/repackage.go` — extracted `repackageOnce(ctx, mag)` (the repackage core, driven by the `rp*` package vars); CLI `repackageRun` now just creates the magazine and calls it. `--runtime` already existed (recorded as a config-artefact annotation via `ConfigCompositionAnnotations`).
- `cmd/parse_models.go` — `parse models <dir>`: `FindModels` → for each model × quant × **model-serving profile** (`modelServingProfiles` filters on `Hardware.IsModelServing()`) sets the `rp*` vars and calls `repackageOnce`. Pure helpers `modelArtifactRepo`/`modelServingProfiles` unit-tested (`cmd/parse_models_test.go`).

**Contract / gotchas:**
- Sources must be **pre-cached** in metahub (like `parse collections`): quant source cache = `models/<name>/<quant>`, artifactType `application/vnd.qnib.model.artifact`. External `quants[].source` auto-cache is a follow-up.
- `repackageOnce` still `logrus.Fatal`s on error → one bad `(quant, profile)` aborts the run; harden to returned errors (tracked).
- `runtime register` (`cmd/runtime.go`) not yet removed — removing it breaks `misc/gocd/bin/test-repackage.sh`; do it with #8 + an e2e update.
- **Knowledge graph is STALE** (`cmd/` changed) — regenerate with `/understand` before relying on `.understand-anything/`.
> Updated after each session that modifies this repo.

## Session: waldur-outfitter-integration — gate-model-serving-on-model-centric (2026-08-24)

### What changed

Companion to the metalib gate (model-serving stack now only fires on
model-centric pulls):

- `cmd/repackage.go` — `warmManipulationCache` now sets
  `ctx = types.WithRequestedModel(ctx, rpName)` +
  `types.WithRequestedQuant(ctx, rpQuantTag)` before
  `ApplyManipulationsToManifests`, so the (now-gated) model-serving stack still
  fires during cache precomputation. Without it the cache would store the
  unmanipulated runtime.
- `misc/gocd/bin/test-repackage.sh` — step 5 now verifies injection via the
  **model-centric pull** (`models/testmodel:q4-km`), not the collection pull.
  Adds `--runtime apps/testmodel-runtime` to the repackage call (so
  `resolveModelCentricRuntime` finds a composition) and pushes the runtime at
  `:latest` (the tag the resolver hardcodes). New step 5a asserts the collection
  pull is NOT injected (gate regression guard).

### Gotchas
- `resolveModelCentricRuntime` hardcodes runtime tag `"latest"` — the runtime
  image the model composes onto must be pushed at `:latest`, regardless of the
  tag in the runtime registration (`runtime register --image ...:1` is only for
  the `/api/v1/runtimes` surface, not the pull path).
- `--runtime` on `repackage` is what creates the composition annotation the
  model-centric resolver reads; without it a `models/<name>:<quant>` pull
  returns nothing (no composition → no rewrite → 404).

## What This Repo Is

Admin and seed tooling for MetaHub. Parses upstream registries
(Docker Hub, Quay, etc.), populates the neo4j graph, and manages
profiles/rules/collections via the metalib magazine interface.

## Architecture

```
cmd/
  main.go                  CLI entrypoint
  version.go               Version command (handles -devN suffix)
  parse.go                 Parse upstream registries
  compute.go               Compute manifestlists, indexes
  seed.go                  Seed profiles and rules
pkg/
  utils/
    version.go             Version struct (Major.Minor.Patch.Dev)
    version_test.go        Version parsing tests
```

## Key Patterns

### Insertion Ordering

When inserting manifests, config and layers **must** be inserted
before `ManifestGetInsert` or the relationships break. This ordering
constraint was discovered and fixed in the fix-oci-conformance session.

### Metalib Dependency

metahub-admin calls metalib functions directly. When metalib renames
functions (like the `ManifestData*` → `Manifest*` rename), metahub-admin
must be updated to match. The 7-function rename in the last session
is the pattern to follow.

### Collection Link Labels

The code must accept both `:ManifestWrapper` and `:Manifest` labels
when linking collections, because legacy data uses the old label.

## Testing

```bash
./test.sh 2>&1 > test.log    # if test.sh exists
go test ./... 2>&1 > test.log  # otherwise
```

## Recent Changes

- 7 function renames (`ManifestData*` → `Manifest*` suffix drop)
- Insertion ordering fix (config/layers before ManifestGetInsert)
- Collection link label fix (accept both ManifestWrapper and Manifest)
- Version parsing updated to handle `-devN` suffix
- SonarQube version auto-detection

## 2026-04-18 (ui-and-manipulation session)

### Session merge

The session branch (`48-ui-and-manipulation`) was cut from `f0e54c1`
(2026-04-13) which predates the merge of `47-pullthrough-repo`
(`5a8cd09`, 2026-04-17). That merge carried `0b662e6`
(`feat(parse-collections): wire Slot, PULLED_BY, unwind, and compute`)
— the exact fix the e2e-dev verify-ce regression guard expects
(Slot + HAS_PUSHED + POINTS_TO->Index + HEAD /v2/.../manifests/<tag>=200).

Merge resolved with these deltas:
- `.bumpversion.cfg` + `cmd/version.go`: bumped to `2.1.0-dev1`
  (session dev bit on top of main's 2.1.0).
- `go.mod`: kept relative `../metahub-core` replace so both host and
  compose-container paths resolve.
- `misc/config/compose-config.yaml`: kept session's `blobs:` scheme +
  added main's `redirect: false`.

### Extended manipulation fix (this session)

`cmd/parse_manipulations.go` now caches upstreams from BOTH legacy
flat `ManipuLayers` AND the new `DefaultConfig + Variants` layer lists
so variant images resolve before `ManipulationFromYAML` runs under the
extended schema. Without this, parsing a migrated CE manipulation
(glow/1.5.0) failed on uncached variant images.

### Verified end-to-end

Against the local compose stack post-merge:

```
parse manifests     → 2 Manifest nodes for qnib/ftest
parse collections   → wires Slot + HAS_PUSHED + PULLED_BY + POINTS_TO->Index
HEAD /v2/qnib/ftest/manifests/latest → 200
(mp:Manipulation)-[:HAS]->(mpl:ManipuLayer) → 2 (legacy compat edge)
```

All three verify-ce failures covered by this session + the merge.

## Gotchas

- **Historical parsed data needs a re-parse.** Before the
  `fix-parse-collections-slot-wiring` change, `parse collections`
  produced the legacy wrapper edges but NOT the Slot-centric edges
  that the OCI `HEAD`/`GET /v2/<repo>/manifests/<tag>` handlers now
  require. Collections imported before the fix (e.g. `qnib/ftest` on
  metahub-dev) show up in `tags/list` but return 404
  `MANIFEST_UNKNOWN` on manifest requests. The fix is forward-only;
  re-run `metahub-admin parse collections <dir>` after seeding
  profiles + rules to reconstruct the missing wiring. No in-place
  migration command is provided.
- **Parse ordering matters.** For pullable output you need (in order):
  `parse profiles` → `parse rules` → `parse collections`. If `generic`
  is not seeded when `parse collections` runs, it will WARN and fall
  back to legacy-edges-only; re-run after seeding fixes it.
- Uses `sonar.login` instead of `sonar.token` in sonar.sh (legacy)
- No `test.sh` wrapper — may need `go test ./...` directly
- Version struct is in `pkg/utils/version.go`, not `internal/`

## Future Direction

The `parse collections` wiring added in the current session is a
tactical fix that delegates to the same metalib helpers the push
path uses. The strategic unification — a single
`MaterializeManifest(Target, Source)` metalib entry point that push,
pullthrough, and parse all funnel through — is documented in
Obsidian at `Metahub/docs/architecture/materialize-manifest-unification.md`
and tracked as a follow-up session. When that lands, parse becomes a
thin adapter that constructs `Target` + `Source` and calls
`MaterializeManifest` — and the Slot/Profile wiring added in this
session moves into the unified materialisation function.

## 2026-04-17 Continuation

### What changed
- **compose-config.yaml updated**: Changed from legacy `configs`+`layers` MinIO buckets to unified `blobs` bucket. Changed from flat string registries (`- "registry:5050"`) to structured `name`/`url` format (`name: local-reg, url: registry:5050`). Aligned with current metalib schema expectations.
- **Admin service removed from compose**: Admin commands now run inside the `dev` container via `go run main.go` from the metahub-admin source mount. This uses the current metalib source, eliminating schema mismatches between admin v1.4.3 and current metalib.

### Gotchas
- The old admin Docker image (v1.4.3) creates `:ManifestData`/`:Layer`/`:Config` labels incompatible with current metalib. Running from source avoids this.
- `compose-config.yaml` must have the `blobs` bucket (not `configs`+`layers`) for the current metalib MinIO code to work.
