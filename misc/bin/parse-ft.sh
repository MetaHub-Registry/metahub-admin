#!/bin/bash

./misc/bin/parse-profiles-rules.sh
go run main.go --config ./misc/config/compose-config.yaml parse manifests /community-edition/collections/qnib/featuretest/zen3.yaml
go run main.go --config ./misc/config/compose-config.yaml parse collections /community-edition/collections/qnib/featuretest/zen3.yaml
go run main.go --config ./misc/config/compose-config.yaml compute manifests
go run main.go --config ./misc/config/compose-config.yaml compute manifestlists
