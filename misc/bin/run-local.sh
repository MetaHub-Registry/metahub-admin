#!/bin/bash
set -e

go run main.go parse --config misc/config/local.yaml profiles  --empty-graph misc/yaml/profiles/library
## Minio Ops
for x in $@;do
	echo "#### Parse Manifest $x"
	go run main.go parse --config misc/config/local.yaml manifests --remove-manifests-alias $x
done

## Neo4j Ops
for x in $@;do
	echo "#### Parse Manifest $x"
	go run main.go parse --config misc/config/local.yaml collections $x
done
echo "#### Compute ManifestList"
go run main.go --config misc/config/local.yaml compute manifestlists