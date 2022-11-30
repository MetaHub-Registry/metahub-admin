#!/bin/bash

CWD=$(pwd)
BIN_PATH=$(dirname "$0")

source ~/gocd_func.sh

cd ${BIN_PATH}

docker-compose -p admin-stack -f docker-compose-data.yaml up -d
sleep 3

MINIO_PORT=$(getMinioPort)
NEO_HTTP_PORT=$(getNeo4jHttpPort)
NEO_BOLT_PORT=$(getNeo4jBoltPort)

checkMinioHealth
checkNeo4jHealth
echo "NEO4J_URL=bolt://localhost:${NEO_BOLT_PORT}" > ${CWD}/test.env
echo "MINIO_URL=localhost:${MINIO_PORT}" >> ${CWD}/test.env
