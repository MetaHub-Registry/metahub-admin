#!/bin/bash
set -e

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

env|sort -u > gocd.env

cat misc/config/metahub-admin.yaml \
	|wildq -M -i yaml '.minio.user = "AWSUSER"' |sed -e "s/AWSUSER/${AWS_ACCESS_KEY}/"  \
	|wildq -M -i yaml '.minio.password = "PASSWD"' |sed -e "s#PASSWD#${AWS_SECRET_ACCESS_KEY}#"  \
	|wildq -M -i yaml '.minio.region = "eu-west-1"' \
	|wildq -M -i yaml '.minio.url = "s3.amazonaws.com"' \
	|wildq -M -i yaml '.minio.configs.bucket = "metahub-prod-configs"' \
	|wildq -M -i yaml '.minio.layers.bucket = "metahub-prod-layers"' \
	|wildq -M -i yaml '.minio.manifests.bucket = "metahub-prod-manifests"' \
	|wildq -M -i yaml '.minio.ssl = true' \
	|wildq -M -i yaml '.neo4j.url = "NEOURL"' |sed -e "s#NEOURL#${NEO4J_URL}#" \
	> ${TMP_DIR}/metahub-admin.yaml

## Fetch Artefact
source ~/gocd.env
curl -sH "Authorization: bearer ${GOCD_TOKEN}" \
     -o ./metahub-admin "${GO_SERVER_URL}/files/${GO_PIPELINE_NAME}/${GO_PIPELINE_COUNTER}/Build/1/Go/metahub-admin"
chmod +x ./metahub-admin

git clone https://gitlab.com/qnib-metahub/community-edition.git ${TMP_DIR}/community-edition

## Render config
export MINIO_URL=${MINIO_URL:-http://localhost:9000}
export NEO4J_URL=${NEO4J_URL:-http://localhost:7687}
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
