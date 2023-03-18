#!/bin/bash
set -ex
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse rules --empty-graph /community-edition/rules/
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse profiles /community-edition/profiles/
if [[ -z ${SKIP_PARSE_MANIFESTS} ]];then
	## Minio Ops
	for x in $@;do
		echo "#### Parse Manifest $x"
		go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse manifests --remove-manifests-alias $x
	done
fi
## Neo4j Ops
for x in $@;do
	echo "#### Parse Manifest $x"
	go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse collections $x
done
echo "#### Compute Manifests"
#go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} compute manifests
echo "#### Compute ManifestList"
#go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} compute manifestlists
