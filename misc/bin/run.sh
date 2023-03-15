#!/bin/sh
set -e

: "${CE_PATH:=/ce}"
: "${CFG_PATH:=./misc/config/compose-config.yaml}"

if [[ -f /etc/metahub/metahub-admin.yaml.tpl ]];then
  gonja-cli /etc/metahub/metahub-admin.yaml.tpl /etc/metahub/metahub-admin.yaml
  export CFG_PATH=/etc/metahub/metahub-admin.yaml
fi

if [[ -z ${1} ]];then
  echo "[ERROR] At least one path to collections needed"
fi

metahub-admin --config ${CFG_PATH} parse rules --empty-graph ${CE_PATH}/rules/
metahub-admin --config ${CFG_PATH} parse profiles ${CE_PATH}/profiles/
if [[ -z ${SKIP_PARSE_MANIFESTS} ]];then
	## Minio Ops
	for x in $@;do
		echo "#### Parse Manifest $x"
		metahub-admin --config ${CFG_PATH} parse manifests --remove-manifests-alias $x
	done
fi
## Neo4j Ops
for x in $@;do
	echo "#### Parse Manifest $x"
	metahub-admin --config ${CFG_PATH} parse collections $x
done
echo "#### Compute Manifests"
metahub-admin --config ${CFG_PATH} compute manifests
echo "#### Compute ManifestList"
metahub-admin --config ${CFG_PATH} compute manifestlists
