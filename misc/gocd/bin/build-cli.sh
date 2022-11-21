#!/bin/bash

cat go.mod|awk '!/^replace/' > go.mod.tmp
mv go.mod.tmp go.mod

TOPDIR=$(git rev-parse --show-toplevel)
git clone https://gitlab-ci-token:${GOCD_TOKEN}@github.com/metahub-registry/metahub-core ext/metalib
go mod edit -replace github.com/metahub-registry/metahub-core=${PWD}/ext/metalib
go build -o ${TOPDIR}/metahub-admin