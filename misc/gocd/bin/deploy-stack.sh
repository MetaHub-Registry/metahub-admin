#!/bin/bash

CWD=$(pwd)
BIN_PATH=$(dirname "$0")

source ~/gocd_func.sh
dumpEnv

cd ${BIN_PATH}

export STACK_NAME=$(echo gocd-${GO_PIPELINE_NAME}_${GO_PIPELINE_COUNTER} |tr '[:upper:]' '[:lower:]')
for x in $(docker ps -q|grep -v $(echo gocd-${GO_STAGE_NAME} |tr '[:upper:]' '[:lower:]'));do docker rm -f ${x};done
docker-compose -p ${STACK_NAME} -f docker-compose-data.yaml up -d
sleep 3

MINIO_PORT=$(getMinioPort)
NEO_HTTP_PORT=$(getNeo4jHttpPort)
NEO_BOLT_PORT=$(getNeo4jBoltPort)

checkMinioHealth
checkNeo4jHealth
echo "NEO4J_URL=bolt://localhost:${NEO_BOLT_PORT}" > ${CWD}/test.env
echo "MINIO_URL=localhost:${MINIO_PORT}" >> ${CWD}/test.env
