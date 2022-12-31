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

if [[ -d ext/metalib ]];then
  git -C ext/metalib pull
else
  git clone https://gitlab-ci-token:${GITLAB_TOKEN}@github.com/metahub-registry/metahub-core ext/metalib
fi

go mod edit -replace github.com/metahub-registry/metahub-core=${PWD}/ext/metalib
go build -o ${TOPDIR}/metahub-admin