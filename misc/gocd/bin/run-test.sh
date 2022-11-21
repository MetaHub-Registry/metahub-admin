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

env|sort -u > gocd.env

if [[ ! -x {TOPDIR}/metahub-admin ]];then
	echo "WARN: No metahub-admin found"
	exit 1
fi

gonja-cli ${TOPDIR}/misc/tpl/metahub-admin.yaml.tpl ${TMP_DIR}/metahub-admin.yaml
## Minio Ops
{TOPDIR}/metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml manifests --remove-manifests-alias  ./misc/yaml/manifests/qnib

## Neo4j Ops
{TOPDIR}/metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml profiles  --empty-graph ./misc/yaml/profiles
{TOPDIR}/metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml collections ./misc/yaml/manifests/qnib
{TOPDIR}/metahub-admin --config ${TMP_DIR}/metahub-admin.yaml compute manifestlists

echo "#### Create SQLite DB"
DBFILE=./metahub.db
{TOPDIR}/metahub-admin --config ${TMP_DIR}/metahub-admin.yaml parse users --metahub-db=${DBFILE} ./misc/yaml/users
{TOPDIR}/metahub-admin --config ${TMP_DIR}/metahub-admin.yaml sqlite --metahub-db=${DBFILE}
rm -rf ${TMP_DIR}
