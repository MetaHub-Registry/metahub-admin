#!/bin/bash
set -e
TOPDIR=$(git rev-parse --show-toplevel)

## Setup metahub-admin config
TMP_DIR=$(mktemp -d -t ci-XXXXXXXXXX)

## Fetch Artefact
source ~/gocd.env
curl -sH "Authorization: bearer ${GOCD_TOKEN}" \
     -o test.env "${GO_SERVER_URL}/files/${GO_PIPELINE_NAME}/${GO_PIPELINE_COUNTER}/${GO_STAGE_NAME}/${GO_STAGE_COUNTER}/Deploy/Setup/misc/gocd/bin/test.env"

source test.env

export MINIO_URL=${MINIO_URL:-http://localhost:9000}
export NEO4J_URL=${NEO4J_URL:-http://localhost:7687}

## Fetch Artefact
curl -sH "Authorization: bearer ${GOCD_TOKEN}" \
     -o metahub-admin "${GO_SERVER_URL}/files/${GO_PIPELINE_NAME}/${GO_PIPELINE_COUNTER}/${GO_STAGE_NAME}/${GO_STAGE_COUNTER}/Build/Go/metahub-admin"
chmod +x metahub-admin

gonja-cli ${TOPDIR}/misc/tpl/metahub-admin.yaml.tpl ${TMP_DIR}/metahub-admin.yaml
## Minio Ops
metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml manifests --remove-manifests-alias  ${TOPDIR}/misc/yaml/manifests/qnib

## Neo4j Ops
metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml profiles  --empty-graph ${TOPDIR}/misc/yaml/profiles
metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml collections ${TOPDIR}/misc/yaml/manifests/qnib
metahub-admin --config ${TMP_DIR}/metahub-admin.yaml compute manifestlists

echo "#### Create SQLite DB"
DBFILE=./metahub.db
metahub-admin --config ${TMP_DIR}/metahub-admin.yaml parse users --metahub-db=${DBFILE} ${TOPDIR}/misc/yaml/users
metahub-admin --config ${TMP_DIR}/metahub-admin.yaml sqlite --metahub-db=${DBFILE}
rm -rf ${TMP_DIR}
