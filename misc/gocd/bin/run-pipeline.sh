#!/bin/bash
set -e

source ~/gocd_func.sh
export STACK_NAME=$(echo gocd-${GO_PIPELINE_NAME}_${GO_PIPELINE_COUNTER} |tr '[:upper:]' '[:lower:]')
dumpEnv

if [[ -f /run/secrets/aws-access-key ]];then
    export AWS_ACCESS_KEY=$(cat /run/secrets/aws-access-key)
elif [[ -f ~/.aws/credentials ]];then
	export AWS_ACCESS_KEY=$(cat ~/.aws/credentials |wildq -M --ini '.s3.aws_access_key_id')
fi
if [[ -f /run/secrets/aws-secret-access-key ]];then
    export AWS_SECRET_ACCESS_KEY=$(cat ca)
elif [[ -f ~/.aws/credentials ]];then
	export AWS_SECRET_ACCESS_KEY=$(cat ~/.aws/credentials |wildq -M --ini '.s3.aws_secret_access_key')
fi

if [[ -f /run/secrets/neo4j-url ]];then
	export NEO4J_URL=$(cat /run/secrets/neo4j-url)
else 
	echo "WARN: No NEO4J_URL found (/run/secrets/neo4j-url), expect 'neo4j' to be available locally"
	NEO4J_URL=bolt://neo4j:7687
fi
## Setup metahub-admin config
TMP_DIR=$(mktemp -d -t ci-XXXXXXXXXX)
TOPDIR=$(git rev-parse --show-toplevel)

## Fetch Artefact
fetchPipelineArtefact "Build" "Go" "metahub-admin"
chmod +x ./metahub-admin

git clone https://gitlab.com/qnib-metahub/community-edition.git ${TMP_DIR}/community-edition
## Render config
fetchPipelineArtefact "Deploy" "Setup" "misc/gocd/bin/test.env"
export $(grep -v '^#' test.env | xargs)
## Set S3 Options
export MINIO_URL=s3.amazonaws.com
export MINIO_BUCKET_BLOBS=metahub-prod-blobs
export MINIO_BUCKET_MANIFESTS=metahub-prod-manifests
export MINIO_REGION=eu-west-1
export MINIO_USER=${AWS_ACCESS_KEY}
export MINIO_PASSWD=${AWS_SECRET_ACCESS_KEY}
export MINIO_SSL=true

gonja-cli ${TOPDIR}/misc/tpl/metahub-admin.yaml.tpl ${TMP_DIR}/metahub-admin.yaml
## Minio Ops
./metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml manifests --remove-manifests-alias  ${TMP_DIR}/community-edition/collections

## Neo4j Ops
./metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml profiles  --empty-graph ${TMP_DIR}/community-edition/profiles
./metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml collections ${TMP_DIR}/community-edition/collections
./metahub-admin --config ${TMP_DIR}/metahub-admin.yaml compute manifestlists

echo "#### Create SQLite DB"
DBFILE=./metahub.db
cp -r ${TOPDIR}/misc/yaml/users ${TMP_DIR}/community-edition/users
./metahub-admin --config ${TMP_DIR}/metahub-admin.yaml parse users --metahub-db=${DBFILE} ${TMP_DIR}/community-edition/users
./metahub-admin --config ${TMP_DIR}/metahub-admin.yaml sqlite --metahub-db=${DBFILE}
rm -rf ${TMP_DIR}
