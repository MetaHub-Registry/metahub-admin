#!/bin/bash
set -e
TOPDIR=$(git rev-parse --show-toplevel)

## Setup metahub-admin config
TMP_DIR=$(mktemp -d -t ci-XXXXXXXXXX)

if [[ ! -f test.env ]];then
	echo "WARN: No test.env found"
	exit 1
fi
source test.env

env|sort -u > gocd-${GO_STAGE_NAME:-test}.env

if [[ ! -x ~/bin/metahub-admin-${GO_PIPELINE_COUNTER} ]];then
	echo "WARN: No metahub-admin (~/bin/metahub-admin-${GO_PIPELINE_COUNTER}) found"
	exit 1
fi

gonja-cli ${TOPDIR}/misc/tpl/metahub-admin.yaml.tpl ${TMP_DIR}/metahub-admin.yaml
## Minio Ops
~/bin/metahub-admin-${GO_PIPELINE_COUNTER} parse --config ${TMP_DIR}/metahub-admin.yaml manifests --remove-manifests-alias  ./misc/yaml/manifests/qnib

## Neo4j Ops
~/bin/metahub-admin-${GO_PIPELINE_COUNTER} parse --config ${TMP_DIR}/metahub-admin.yaml profiles  --empty-graph ./misc/yaml/profiles
~/bin/metahub-admin-${GO_PIPELINE_COUNTER} parse --config ${TMP_DIR}/metahub-admin.yaml collections ./misc/yaml/manifests/qnib
~/bin/metahub-admin-${GO_PIPELINE_COUNTER} --config ${TMP_DIR}/metahub-admin.yaml compute manifestlists

echo "#### Create SQLite DB"
DBFILE=./metahub.db
~/bin/metahub-admin-${GO_PIPELINE_COUNTER} --config ${TMP_DIR}/metahub-admin.yaml parse users --metahub-db=${DBFILE} ./misc/yaml/users
~/bin/metahub-admin-${GO_PIPELINE_COUNTER} --config ${TMP_DIR}/metahub-admin.yaml sqlite --metahub-db=${DBFILE}
rm -rf ${TMP_DIR}
