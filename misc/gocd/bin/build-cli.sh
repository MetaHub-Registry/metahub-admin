#!/bin/bash

if [[ -f ~/gocd_func.sh ]];then
   FUNC_SRC=~/gocd_func.sh
elif [[ -f /opt/qnib/gocd/src/gocd_func.sh ]];then
   FUNC_SRC=/opt/qnib/gocd/src/gocd_func.sh
else 
    echo "ERROR: No gocd_func.sh found"
    exit 1
fi
source ${FUNC_SRC}
dumpEnv

cat go.mod|awk '!/^replace/' > go.mod.tmp
mv go.mod.tmp go.mod
mkdir -p ~/bin
TOPDIR=$(git rev-parse --show-toplevel)

if [[ -f ~/gocd.env ]];then
    source ~/gocd.env
fi

if [[ -z ${GITLAB_REPO_RO} ]];then
    echo "GITLAB_REPO_RO not set"
    exit 1
fi

if [[ -f metalib.tar ]];then
   echo "> Extracting metalib.tar"
   mkdir -p ext
   tar -xf metalib.tar -C ext
elif [[ "${DEPLOYMENT}" == "dev" ]];then
   echo "> git clone --branch=develop https://gitlab-ci-token:GITLAB_REPO_RO@github.com/metahub-registry/metahub-core ext/metalib"
   git clone --branch=develop https://gitlab-ci-token:${GITLAB_REPO_RO}@github.com/metahub-registry/metahub-core ext/metalib
elif [[ "${DEPLOYMENT}" == "prod" ]];then
   echo "> git clone --branch=main https://gitlab-ci-token:GITLAB_REPO_RO@github.com/metahub-registry/metahub-core ext/metalib"
   git clone --branch=main https://gitlab-ci-token:${GITLAB_REPO_RO}@github.com/metahub-registry/metahub-core ext/metalib
else 
   echo "ERROR: No DEPLOYMENT branch set"
   exit 1
fi
go mod edit -replace github.com/metahub-registry/metahub-core=${PWD}/ext/metalib
go mod tidy

go build -o ${TOPDIR}/metahub-admin