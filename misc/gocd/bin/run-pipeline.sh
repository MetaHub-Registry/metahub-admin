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

## Setup metahub-admin config
TMP_DIR=$(mktemp -d -t ci-XXXXXXXXXX)

NEO4J_URL=bolt://portainer-swarm.fritz.box:17687
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





git clone https://gitlab.com/qnib-metahub/community-edition.git ${TMP_DIR}/community-edition
## Minio Ops
metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml manifests --remove-manifests-alias  ${TMP_DIR}/community-edition/manifests

## Neo4j Ops
metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml profiles  --empty-graph ${TMP_DIR}/community-edition/profiles
metahub-admin parse --config ${TMP_DIR}/metahub-admin.yaml collections ${TMP_DIR}/community-edition/manifests
metahub-admin --config ${TMP_DIR}/metahub-admin.yaml compute manifestlists

echo "#### Create SQLite DB"
DBFILE=./metahub.db
cp -r ./misc/yaml/users ${TMP_DIR}/community-edition/users
metahub-admin --config ${TMP_DIR}/metahub-admin.yaml parse users --metahub-db=${DBFILE} ./misc/yaml/users
metahub-admin --config ${TMP_DIR}/metahub-admin.yaml sqlite --metahub-db=${DBFILE}
rm -rf ${TMP_DIR}
