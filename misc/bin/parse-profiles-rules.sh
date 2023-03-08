#!/bin/bash
set -ex
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse rules --empty-graph --empty-minio /community-edition/rules/os/linux.yaml
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse rules /community-edition/rules/cpu/generic.yaml
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse profiles /community-edition/profiles/generic/cpu.yaml
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse profiles /community-edition/profiles/generic.yaml
