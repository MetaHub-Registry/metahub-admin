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

- 7 function renames (`ManifestData*` → `Manifest*` suffix drop)
- Insertion ordering fix (config/layers before ManifestGetInsert)
- Collection link label fix (accept both ManifestWrapper and Manifest)
- Version parsing updated to handle `-devN` suffix
- SonarQube version auto-detection

## Gotchas

- Uses `sonar.login` instead of `sonar.token` in sonar.sh (legacy)
- No `test.sh` wrapper — may need `go test ./...` directly
- Version struct is in `pkg/utils/version.go`, not `internal/`
