#!/bin/bash

source ~/gocd_func.sh
export STACK_NAME=$(echo gocd-${GO_JOB_NAME}-${GO_PIPELINE_COUNTER} |tr '[:upper:]' '[:lower:]')
dumpEnv

### Copy /minio to current dir and upload to artefacts
docker-compose -p ${STACK_NAME} -f docker-compose-data.yaml down -v
