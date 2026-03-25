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

if [[ -f metalib_result/result.yaml ]];then
   echo ">> found metalib_result.yml"
   echo "####"
   cat metalib_result/result.yaml
   echo "####"
   DOCKERFILE="Dockerfile.result"
   METALIB_IMAGE=$(yq -r .image metalib_result/result.yaml)
   cat >> result.yaml << EOF
metalib:
   image: ${METALIB_IMAGE}
EOF
else
   echo "No metalib_result/result.yaml to get the metalib image"
   exit 1
fi
set -ex
docker build . --build-arg=METALIB_IMAGE=METALIB_IMAGE -t test --target=cbuilder
docker run -d --name target test sleep 30
docker cp target:/go/metahub-admin/metahub-admin .
docker rm -f target

go mod edit -replace github.com/metahub-registry/metahub-core=${PWD}/ext/metalib
go mod tidy

go build -o ${TOPDIR}/metahub-admin