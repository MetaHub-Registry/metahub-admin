#!/bin/bash
set -e
export STACK_NAME=$(echo gocd-${GO_STAGE_NAME}-${GO_PIPELINE_COUNTER} |tr '[:upper:]' '[:lower:]')
TOPDIR=$(git rev-parse --show-toplevel)
## Setup metahub-admin config
TMP_DIR=$(mktemp -d -t ci-XXXXXXXXXX)

## Fetch Artefact
source ~/gocd.env
if [[ ! -f test.env ]];then
  curl -sH "Authorization: bearer ${GOCD_TOKEN}" \
     -o test.env "${GO_SERVER_URL}/files/${GO_PIPELINE_NAME}/${GO_PIPELINE_COUNTER}/Deploy/1/Setup/misc/gocd/bin/test.env"
fi
source test.env
echo ">> MINIO_URL: ${MINIO_URL}"
echo ">> NEO4J_URL: ${NEO4J_URL}"
export MINIO_URL=${MINIO_URL:-http://localhost:9000}
export NEO4J_URL=${NEO4J_URL:-http://localhost:7687}

## Fetch Artefact
curl -sH "Authorization: bearer ${GOCD_TOKEN}" \
     -o ./metahub-admin "${GO_SERVER_URL}/files/${GO_PIPELINE_NAME}/${GO_PIPELINE_COUNTER}/Build/1/Go/metahub-admin"
chmod +x ./metahub-admin

gonja-cli ${TOPDIR}/misc/tpl/metahub-admin.yaml.tpl ${TMP_DIR}/metahub-admin.yaml
## Minio Ops
./metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml manifests --remove-manifests-alias  ${TOPDIR}/misc/yaml/manifests/qnib

## Neo4j Ops
./metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml profiles  --empty-graph ${TOPDIR}/misc/yaml/profiles
./metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml collections ${TOPDIR}/misc/yaml/manifests/qnib
./metahub-admin --config ${TMP_DIR}/metahub-admin.yaml compute manifestlists

echo "#### Create SQLite DB"
DBFILE=./metahub.db
./metahub-admin --config ${TMP_DIR}/metahub-admin.yaml parse users --metahub-db=${DBFILE} ${TOPDIR}/misc/yaml/users
./metahub-admin --config ${TMP_DIR}/metahub-admin.yaml sqlite --metahub-db=${DBFILE}
rm -rf ${TMP_DIR}
