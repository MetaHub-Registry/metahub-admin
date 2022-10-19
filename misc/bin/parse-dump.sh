#!/bin/sh

YAMLPATH=$1
if [ -z "$YAMLPATH" ]; then
    echo "Usage: $0 <PATH TO YAML DIR>"
    echo "  Within the YAML dir, there should be a directories 'profiles' and 'manifests'"
    exit 1
fi
if [ ! -d "$YAMLPATH" ]; then
    echo "No such directory: $YAMLPATH"
    exit 1
elif [ ! -d "$YAMLPATH/profiles" ]; then
    echo "No such directory: $YAMLPATH/profiles"
    exit 1
elif [ ! -d "$YAMLPATH/manifests" ]; then
    echo "No such directory: $YAMLPATH/manifests"
    exit 1
fi

TOUT=30
echo "### Wait for CE to be ready"
while [ ! -f "${YAMLPATH}/.done" ]; do
    echo -n "."
    sleep 1
    TOUT=$((TOUT-1))
    if [ $TOUT -eq 0 ]; then
        echo "Timeout"
        exit 1
    fi
done
echo " CE ready in ${YAMLPATH}"

set -x
## Minio Ops
metahub-admin parse --config /go/metahub-admin/misc/config/compose-config.yaml manifests --empty-configs --empty-manifests $YAMLPATH/manifests

## Neo4j Ops
metahub-admin parse --config /go/metahub-admin/misc/config/compose-config.yaml profiles  --empty-graph $YAMLPATH/profiles
metahub-admin parse --config /go/metahub-admin/misc/config/compose-config.yaml collections $YAMLPATH/manifests
metahub-admin --config /go/metahub-admin/misc/config/compose-config.yaml compute manifestlists

echo "#### Create SQLite DB"
DBFILE=/go/metahub-admin/metahub.db
metahub-admin --config /go/metahub-admin/misc/config/compose-config.yaml parse users --metahub-db=${DBFILE} $YAMLPATH/misc/yaml/users
metahub-admin --config /go/metahub-admin/misc/config/compose-config.yaml sqlite --metahub-db=${DBFILE}