#!/bin/bash
set -ex
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse rules --empty-graph=keep-upstream \
	/community-edition/rules/os/linux.yaml \
	/community-edition/rules/cpu/noop.yaml
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse profiles \
	/community-edition/profiles/generic.yaml
if [[ -z ${SKIP_PARSE_MANIFESTS} ]];then
	## Minio Ops
	echo "#### Parse Manifest $@"
	go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse manifests $@
fi
## Neo4j Ops
echo "#### Parse Manifest $x"
echo ">> go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse collections $@"
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} parse collections $@

echo "#### Compute Manifests"
echo ">> go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} compute manifests"
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} compute manifests
echo "#### Compute ManifestList"
echo ">> go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} compute manifestlists"
go run main.go --config ${CFG_PATH:-./misc/config/compose-config.yaml} compute manifestlists
