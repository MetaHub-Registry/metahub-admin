#!/bin/sh

YAMLPATH=$1
if [ -z "$YAMLPATH" ]; then
    echo "Usage: $0 <PATH TO YAML DIR>"
    echo "  Within the YAML dir, there should be a directories 'profiles' and 'collections'"
    exit 1
fi
if [ ! -d "$YAMLPATH" ]; then
    echo "No such directory: $YAMLPATH"
    exit 1
elif [ ! -d "$YAMLPATH/profiles" ]; then
    echo "No such directory: $YAMLPATH/profiles"
    exit 1
elif [ ! -d "$YAMLPATH/collections" ]; then
    echo "No such directory: $YAMLPATH/collections"
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

set -xe
## Minio Ops
metahub-admin parse --config ./misc/config/compose-config.yaml manifests --remove-manifests-alias  $YAMLPATH/collections

## Neo4j Ops
metahub-admin parse --config ./misc/config/compose-config.yaml profiles  --empty-graph $YAMLPATH/profiles
metahub-admin parse --config ./misc/config/compose-config.yaml collections $YAMLPATH/collections
metahub-admin --config ./misc/config/compose-config.yaml compute manifestlists

echo "#### Create SQLite DB"
DBFILE=./metahub.db
cp -r ./misc/yaml/users /opt/community-edition/
metahub-admin --config ./misc/config/compose-config.yaml parse users --metahub-db=${DBFILE} $YAMLPATH/users
metahub-admin --config ./misc/config/compose-config.yaml sqlite --metahub-db=${DBFILE}