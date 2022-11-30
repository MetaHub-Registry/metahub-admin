#!/bin/bash

source ~/gocd_func.sh
export STACK_NAME=$(echo gocd-${GO_JOB_NAME}-${GO_PIPELINE_COUNTER} |tr '[:upper:]' '[:lower:]')
dumpEnv

### Copy /minio to current dir and upload to artefacts
CNT_NAME=$(docker ps| grep ${STACK_NAME} | awk '/backup/{print $NF}')
docker exec -t --workdir /minio ${CNT_NAME} tar cf /backup/minio.tar .
