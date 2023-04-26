#!/bin/bash

source ~/gocd_func.sh
dumpEnv

cat go.mod|awk '!/^replace/' > go.mod.tmp
mv go.mod.tmp go.mod
mkdir -p ~/bin
TOPDIR=$(git rev-parse --show-toplevel)

if [[ -f ~/gocd.env ]];then
    source ~/gocd.env
fi

if [[ -z ${GITLAB_TOKEN} ]];then
    echo "GITLAB_TOKEN not set"
    exit 1
fi

if [[ "${DEPLOYMENT}" == "dev" ]];then
   echo "> git clone --branch=develop https://gitlab-ci-token:GITLAB_TOKEN@github.com/metahub-registry/metahub-core ext/metalib"
   git clone --branch=develop https://gitlab-ci-token:${GITLAB_TOKEN}@github.com/metahub-registry/metahub-core ext/metalib
elif [[ "${DEPLOYMENT}" == "prod" ]];then
   echo "> git clone --branch=main https://gitlab-ci-token:GITLAB_TOKEN@github.com/metahub-registry/metahub-core ext/metalib"
   git clone --branch=main https://gitlab-ci-token:${GITLAB_TOKEN}@github.com/metahub-registry/metahub-core ext/metalib
else 
   echo "ERROR: No DEPLOYMENT branch set"
   exit 1
fi
go mod edit -replace github.com/metahub-registry/metahub-core=${PWD}/ext/metalib
go mod tidy

go build -o ${TOPDIR}/metahub-admin