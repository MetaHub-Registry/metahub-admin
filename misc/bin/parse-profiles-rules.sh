#!/bin/bash

MHA=$(which metahub-admin)

if [[ $? -ne 0 || ! -z "${RUN_GO}" ]];then
	echo "metahub-admin not found"
	MHA="go run main.go"
fi

set -ex
${MHA} --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse --empty-graph=${EMPTY_GRAPH:-keep-upstream} rules \
	/community-edition/rules/cpu/zen.yaml \
	/community-edition/rules/cpu/zen2.yaml \
	/community-edition/rules/cpu/zen3.yaml \
	/community-edition/rules/os/linux.yaml /community-edition/rules/cpu/noop.yaml 
${MHA} --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse profiles \
	/community-edition/profiles/generic.yaml \
	/community-edition/profiles/cpu/zen3.yaml \
	/community-edition/profiles/cpu/zen2.yaml \
	/community-edition/profiles/cpu/zen.yaml

if [[ -z ${1} ]];then
    echo "#### No collection to parse"
    exit 0
fi
if [[ -z ${SKIP_PARSE_MANIFESTS} ]];then
	## Minio Ops
	echo "#### Parse Manifest $@"
	${MHA} --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse manifests $@
fi
## Neo4j Ops
echo "#### Parse Manifest $x"
echo ">> ${MHA} --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse collections $@"
${MHA} --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse collections $@

echo "#### Compute Manifests"
echo ">> go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} compute manifests"
${MHA} --config ${CFG_PATH:-./misc/config/compose-config.yaml} compute manifests
echo "#### Compute ManifestList"
echo ">> go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} compute manifestlists"
${MHA} --config ${CFG_PATH:-./misc/config/compose-config.yaml} compute manifestlists
