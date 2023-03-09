#!/bin/bash

go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse rules --empty-graph ${1} /community-edition/rules/os/linux.yaml
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse rules /community-edition/rules/cpu/noop.yaml
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse profiles /community-edition/profiles/generic.yaml
### skylake_avx512
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse rules /community-edition/rules/cpu/skylake_avx512.yaml
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse profiles /community-edition/profiles/cpu/skylake_avx512.yaml
### g5
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse rules /community-edition/rules/cpu/x86_64_v3.yaml
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse rules /community-edition/rules/gpu/nvidia.yaml
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse profiles /community-edition/profiles/node/aws/g5.yaml
### gromacs/tmpi/2021.5
go run main.go --config ./misc/config/compose-config.yaml parse manifests /community-edition/collections/gromacs/tmpi/2021.5.yaml
go run main.go --config ./misc/config/compose-config.yaml parse collections /community-edition/collections/gromacs/tmpi/2021.5.yaml
go run main.go --config ./misc/config/compose-config.yaml compute manifests
go run main.go --config ./misc/config/compose-config.yaml compute manifestlists
