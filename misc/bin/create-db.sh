#!/bin/bash

DBFILE=$1
if [ -z "$DBFILE" ]; then
    echo "Usage: $0 <dbfile>"
    exit 1
fi
if [ -f "$DBFILE" ]; then
    read -r -p "File '${DBFILE} already exist, are you sure? [y/N] " response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]
    then
        rm -f $1
    fi
fi

go run main.go --config misc/config/metahub-admin.yaml parse users --user-db=${DBFILE} misc/yaml/users
