#!/bin/bash

source ~/gocd_func.sh
export STACK_NAME=$(echo gocd-${GO_STAGE_NAME}-${GO_PIPELINE_COUNTER} |tr '[:upper:]' '[:lower:]')
dumpEnv

set -x

### Copy /minio to current dir and upload to artefacts
CNT_NAME=$(docker ps| grep ${STACK_NAME} | awk '/backup/{print $NF}')
if [[ -z ${CNT_NAME} ]];then
    echo "No container with name 'backup' found"
    exit 1
fi

docker exec -t --workdir /minio ${CNT_NAME} tar cf /backup/minio.tar .
