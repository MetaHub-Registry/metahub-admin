#!/bin/bash

DBFILE=$1
YAMLDIR=$2
if [ -z "$DBFILE" ] && [ -z $YAMLDIR ]; then
    echo "Usage: $0 <dbfile> <yaml_dir>/(users|profiles)"
    exit 1
fi

if [ -f "$DBFILE" ]; then
    read -r -p "File '${DBFILE} already exist, are you sure? [y/N] " response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]
    then
        rm -f $1
    fi
fi
go run main.go --config misc/config/local.yaml parse users --metahub-db=${DBFILE} ${YAMLDIR}/users
go run main.go --config misc/config/local.yaml parse profiles --metahub-db=${DBFILE} ${YAMLDIR}/profiles
go run main.go --config misc/config/local.yaml sqlite --metahub-db=${DBFILE}