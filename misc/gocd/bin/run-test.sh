#!/bin/bash
set -e

## Setup metahub-admin config
TMP_DIR=$(mktemp -d -t ci-XXXXXXXXXX)

if [[ ! -f test.env ]];then
	echo "WARN: No test.env found"
	exit 1
fi
source test.env

env|sort -u > gocd.env

gonja-cli misc/tpl/metahub-admin.yaml.tpl ${TMP_DIR}/metahub-admin.yaml
## Minio Ops
metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml manifests --remove-manifests-alias  ./misc/yaml/manifests/qnib

## Neo4j Ops
metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml profiles  --empty-graph ./misc/yaml/profiles
metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml collections ./misc/yaml/manifests/qnib
metahub-admin --config ${TMP_DIR}/metahub-admin.yaml compute manifestlists

echo "#### Create SQLite DB"
DBFILE=./metahub.db
metahub-admin --config ${TMP_DIR}/metahub-admin.yaml parse users --metahub-db=${DBFILE} ./misc/yaml/users
metahub-admin --config ${TMP_DIR}/metahub-admin.yaml sqlite --metahub-db=${DBFILE}
rm -rf ${TMP_DIR}
