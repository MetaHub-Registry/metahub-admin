#!/bin/bash

TOPDIR=$(git rev-parse --show-toplevel)
git clone https://gitlab-ci-token:${GOCD_TOKEN}@github.com/metahub-registry/metahub-api ext/metalib
go mod edit -replace github.com/metahub-registry/metahub-core=${PWD}/ext/metalib
go build -o ${TOPDIR}/metahub-admin