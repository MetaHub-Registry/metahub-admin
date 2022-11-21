#!/bin/bash

env|sort -u > gocd-${GO_STAGE_NAME:-build}.env

cat go.mod|awk '!/^replace/' > go.mod.tmp
mv go.mod.tmp go.mod
mkdir -p ~/bin
TOPDIR=$(git rev-parse --show-toplevel)
git clone https://gitlab-ci-token:${GOCD_TOKEN}@github.com/metahub-registry/metahub-core ext/metalib
go mod edit -replace github.com/metahub-registry/metahub-core=${PWD}/ext/metalib
go build -o ${TOPDIR}/metahub-admin