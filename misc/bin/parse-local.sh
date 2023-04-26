#!/bin/bash
set -ex

go run main.go parse --config ${CFG_PATH:-misc/config/local.yaml} profiles  --empty-graph=keep-upstream ${PROFILE_PATH:-misc/yaml/profiles/library}
## Minio Ops
for x in $@;do
	echo "#### Parse Manifest $x"
	go run main.go parse --config ${CFG_PATH:-misc/config/local.yaml} manifests $x
done

## Neo4j Ops
for x in $@;do
	echo "#### Parse Collection $x"
	go run main.go parse --config ${CFG_PATH:-misc/config/local.yaml} collections $x
done