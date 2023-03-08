#!/bin/bash

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
elif [ ! -d "$YAMLPATH/collections" ]; then
    echo "No such directory: $YAMLPATH/collections"
    exit 1
fi
set -x
go run main.go parse --config ./misc/config/metahub-admin.yaml manifests $YAMLPATH/collections

go run main.go parse --config ./misc/config/metahub-admin.yaml profiles $YAMLPATH/profiles
go run main.go parse --config ./misc/config/metahub-admin.yaml collections $YAMLPATH/collections
go run main.go --config misc/config/metahub-admin.yaml compute manifestlists