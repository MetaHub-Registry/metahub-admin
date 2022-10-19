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
elif [ ! -d "$YAMLPATH/manifests" ]; then
    echo "No such directory: $YAMLPATH/manifests"
    exit 1
fi
set -x
go run main.go parse --config ./misc/config/metahub-admin.yaml manifests $YAMLPATH/manifests

go run main.go parse --config ./misc/config/metahub-admin.yaml profiles $YAMLPATH/profiles
go run main.go parse --config ./misc/config/metahub-admin.yaml collections $YAMLPATH/manifests
go run main.go --config misc/config/metahub-admin.yaml compute manifestlists