#!/bin/bash
set -e 
: "${CFG_PATH:=./misc/config/compose-config.yaml}"
: "${EMPTY_GRAPH:=keep-upstream}"

YAMLDIR=$1
if [ -z $YAMLDIR ]; then
    echo "Usage: $0 <yaml_dir> <manifest_dir>"
    exit 1
fi
set -ex

go run main.go --config ${CFG_PATH} parse rules --empty-graph=${EMPTY_GRAPH}  ${YAMLDIR}/rules
go run main.go --config ${CFG_PATH} parse manipulations ${YAMLDIR}/manipulations
go run main.go --config ${CFG_PATH} parse profiles ${YAMLDIR}/profiles

go run main.go --config ${CFG_PATH} parse manifests $2
go run main.go --config ${CFG_PATH} parse collections $2
go run main.go --config ${CFG_PATH} compute manifests
go run main.go --config ${CFG_PATH} compute manifestlists
