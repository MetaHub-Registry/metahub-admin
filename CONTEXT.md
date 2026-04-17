# Context: metahub-admin

> This file helps AI agents understand the repo quickly.
> Updated after each session that modifies this repo.

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

- **Slot wiring in `parse collections`** (`fix-parse-collections-slot-wiring`)
  — after each parsed manifest is linked to its platform, parse now
  writes the Slot-centric edges that the OCI pull path requires:
  `:Slot{repo,tag,generic}` + `HAS_PUSHED` + `PULLED_BY` +
  `ProfileUnwindRules`. After the per-collection loop finishes,
  `ComputeManifestlistsForRepoTag` runs to build the per-profile
  `:Index`. Legacy `Repo/Tag-[:UPSTREAM]->Manifest` edges are preserved
  for backward compatibility.
- 7 function renames (`ManifestData*` → `Manifest*` suffix drop)
- Insertion ordering fix (config/layers before ManifestGetInsert)
- Collection link label fix (accept both ManifestWrapper and Manifest)
- Version parsing updated to handle `-devN` suffix
- SonarQube version auto-detection

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
