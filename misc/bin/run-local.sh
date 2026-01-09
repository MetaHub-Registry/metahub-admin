#!/bin/bash
set -ex
if $(command -v go &> /dev/null); then
	CMD="go run main.go"
else
	CMD="metahub-admin"
fi

$CMD --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse rules --empty-graph=${EMPTY_GRAPH:-none} /community-edition/rules/
$CMD --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse profiles /community-edition/profiles/
$CMD --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse manipulations /community-edition/manipulations/layer/glow/
## Manipulations
if [[ -z ${SKIP_PARSE_MANIFESTS} ]];then
	## Minio Ops
	for x in $@;do
		echo "#### Parse Manifest $x"
		$CMD --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse manifests $x
	done
fi
## Neo4j Ops
for x in $@;do
	echo "#### Parse Manifest $x"
	$CMD --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse collections $x
done
echo "#### Compute Manifests"
$CMD --config ${CFG_PATH:-./misc/config/compose-config.yaml} compute manifests
echo "#### Compute ManifestList"
$CMD --config ${CFG_PATH:-./misc/config/compose-config.yaml} compute manifestlists