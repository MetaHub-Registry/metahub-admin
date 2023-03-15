#!/bin/bash
set -ex
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse rules --empty-graph ${1} /community-edition/rules/os/linux.yaml
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse rules /community-edition/rules/cpu/noop.yaml
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse rules /community-edition/rules/cpu/zen3.yaml
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse profiles /community-edition/profiles/generic.yaml
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse profiles /community-edition/profiles/cpu/zen3.yaml
