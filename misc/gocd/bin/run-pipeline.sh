#!/bin/bash
set -e

source ~/gocd_func.sh

dumpEnv

if [[ -f /run/secrets/aws-access-key ]];then
    export AWS_ACCESS_KEY=$(cat /run/secrets/aws-access-key)
elif [[ -f ~/.aws/credentials ]];then
	export AWS_ACCESS_KEY=$(cat ~/.aws/credentials |wildq -M --ini '.s3.aws_access_key_id')
fi
if [[ -f /run/secrets/aws-secret-access-key ]];then
    export AWS_SECRET_ACCESS_KEY=$(cat /run/secrets/aws-secret-access-key)
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
gonja-cli ${TOPDIR}/misc/tpl/metahub-admin.yaml.tpl ${TMP_DIR}/metahub-admin.yaml
## Minio Ops
./metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml manifests --remove-manifests-alias  ${TMP_DIR}/community-edition/manifests

## Neo4j Ops
./metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml profiles  --empty-graph ${TMP_DIR}/community-edition/profiles
./metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml collections ${TMP_DIR}/community-edition/manifests
./metahub-admin --config ${TMP_DIR}/metahub-admin.yaml compute manifestlists

echo "#### Create SQLite DB"
DBFILE=./metahub.db
cp -r ./misc/yaml/users ${TMP_DIR}/community-edition/users
./metahub-admin --config ${TMP_DIR}/metahub-admin.yaml parse users --metahub-db=${DBFILE} ./misc/yaml/users
./metahub-admin --config ${TMP_DIR}/metahub-admin.yaml sqlite --metahub-db=${DBFILE}
rm -rf ${TMP_DIR}
