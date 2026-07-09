#!/bin/bash
# End-to-end test for `metahub-admin repackage` against the oras-conformance
# compose stack (deploy-test.sh stands up metahub-api + minio + neo4j).
#
# Steps (as commissioned):
#   1. push a two-file (250 MB each) ORAS model artifact to metahub
#   2. run `metahub-admin repackage` -> a two-layer OCI model-layer image
#   3. pull the image manifest and inspect the layers (assert 2 tar layers)
#
# Convention mirrors run-oras-conformance.sh:
#   - metahub IS the registry, reachable at ${MH_URL:-localhost:8080} (plain http)
#   - push user `oci`, password ${OCI_MH_PASS:-oci}
#
# `metahub-admin repackage` also needs direct MinIO + Neo4j access (to read the
# pushed artifact and persist the manipulation). Those run as compose containers
# with host-mapped random ports, so we discover them with `docker port` and write
# a matching ~/.metahub-admin.yaml (same minio:/neo4j: shape as metahub-api.yaml).
set -euo pipefail

MH_URL="${MH_URL:-localhost:8080}"          # metahub-api registry endpoint
MH_USER="oci"
MH_PASS="${OCI_MH_PASS:-oci}"
# Attach the model manipulation to this hardware-alias profile `bazzite`
# (an RTX 3060 / 12 GB host; ships in community-edition as profiles/bazzite.yaml).
# The profile MUST carry a hardware.gpu block → model-serving → the pull-path
# resolver runs (capacity from hardware.gpu.vram_gb). The test seeds it (with a
# hardware block) below so the run is self-contained.
MH_PROFILE="${MH_PROFILE:-bazzite}"
ADMIN="${MH_ADMIN:-./metahub-admin}"

NAME="testmodel"
QUANT="q4-km"
ARTIFACT_REPO="models/${NAME}"
ARTIFACT_TYPE="application/vnd.qnib.model.artifact"
ARTIFACT_REF="${QUANT}"
LAYER_IMAGE="models/${NAME}-layer:${QUANT}"

# --- discover the compose stack's minio/neo4j host ports -------------------
# deploy-test.sh names the stack: gocd-<GO_JOB_NAME>-<GO_PIPELINE_COUNTER> (lc).
STACK="$(echo "gocd-${GO_JOB_NAME:-job}-${GO_PIPELINE_COUNTER:-1}" | tr '[:upper:]' '[:lower:]')"
portOf() { # <service> <container-port>
  docker port "${STACK}-$1-1" "$2" 2>/dev/null | head -1 | sed 's/.*://'
}
MINIO_HP="$(portOf minio 9000)"
NEO_HP="$(portOf neo4j 7687)"
echo "== stack=${STACK} minio_host_port=${MINIO_HP} neo4j_bolt_host_port=${NEO_HP} =="
[ -n "${MINIO_HP}" ] || { echo "FAIL: could not find minio host port for ${STACK}-minio-1"; docker ps --format '{{.Names}}'; exit 1; }
[ -n "${NEO_HP}" ]   || { echo "FAIL: could not find neo4j host port for ${STACK}-neo4j-1"; docker ps --format '{{.Names}}'; exit 1; }

# --- metahub-admin config pointing at the stack (mirrors metahub-api.yaml) ---
CFG="$(pwd)/mh-admin.yaml"
cat > "${CFG}" <<EOF
minio:
  url: localhost:${MINIO_HP}
  region: us-east-1
  user: minio
  password: miniominio
  redirect: false
  ssl: false
  manifests:
    bucket: manifests
  blobs:
    bucket: blobs
neo4j:
  dbname: neo4j
  user: neo4j
  password: test
  url: bolt://localhost:${NEO_HP}
EOF
echo "== wrote ${CFG} =="

work="$(mktemp -d)"; trap 'rm -rf "$work"' EXIT

echo "== 0. seed the ${MH_PROFILE} profile so repackage can attach to it =="
# Mirrors community-edition/profiles/12gb/bazzite.yaml. A subdir profile takes the
# label "<dir>#<file>" (e.g. vram#12gb), so split MH_PROFILE on '#' to lay the
# file out accordingly; parse it so ProfileGetByLabel(${MH_PROFILE}) resolves.
# (A '#'-less MH_PROFILE seeds as a top-level file → label == filename.)
if [[ "${MH_PROFILE}" == *#* ]]; then
  prof_path="${MH_PROFILE%%#*}/${MH_PROFILE#*#}"
else
  prof_path="${MH_PROFILE}"
fi
mkdir -p "${work}/profiles/$(dirname "${prof_path}")"
# A hardware.gpu block makes the profile model-serving (IsModelServing) so the
# dynamic model-resolution path runs; capacity (vram_gb) drives quant-fit.
cat > "${work}/profiles/${prof_path}.yaml" <<'PRF'
rule: os:linux=true
hardware:
  gpu:
    vendor: nvidia
    vram_gb: 12
PRF
"${ADMIN}" --config "${CFG}" parse profiles "${work}/profiles"

echo "== 1. build + push a two-file (250 MB each) ORAS model artifact =="
# Two same-quant shard files; leading GGUF magic + .gguf title so the repackage
# identifier recognises them (title precedence; magic-byte as belt-and-braces).
files=()
for i in 1 2; do
  f="${NAME}-Q4_K_M-0000${i}-of-00002.gguf"
  { printf 'GGUF'; head -c $((250*1024*1024 - 4)) /dev/urandom; } > "${work}/${f}"
  files+=("${f}")
done
( cd "${work}" && oras push --plain-http \
    --artifact-type "${ARTIFACT_TYPE}" \
    -u "${MH_USER}" -p "${MH_PASS}" \
    "${MH_URL}/${ARTIFACT_REPO}:${ARTIFACT_REF}" \
    "${files[0]}" "${files[1]}" )
echo "   pushed ${MH_URL}/${ARTIFACT_REPO}:${ARTIFACT_REF} (2 x 250 MB)"

echo "== 2. metahub-admin repackage -> two-layer model-layer image =="
"${ADMIN}" --config "${CFG}" repackage \
  --artifact-repo "${ARTIFACT_REPO}" \
  --artifact-type "${ARTIFACT_TYPE}" \
  --name "${NAME}" --quant-tag "${QUANT}" --profile "${MH_PROFILE}" \
  --registry-url "${MH_URL}" \
  --registry-user "${MH_USER}" --registry-pass "${MH_PASS}" \
  --registry-tls=false
echo "   repackaged -> ${LAYER_IMAGE}"

echo "== 3. pull the model-layer manifest and inspect the layers =="
LAYER_REPO="models/${NAME}-layer"
mani="$(oras manifest fetch --plain-http -u "${MH_USER}" -p "${MH_PASS}" \
        "${MH_URL}/${LAYER_IMAGE}")"
echo "${mani}" | jq .
# metahub serves a tag as an OCI image index (Slot->Index->Manifest aggregation);
# follow it to the child image manifest that actually carries the layers.
if echo "${mani}" | jq -e '.mediaType | test("image.index|manifest.list")' >/dev/null 2>&1; then
  child="$(echo "${mani}" | jq -r '.manifests[0].digest')"
  echo "   index detected → resolving child manifest ${child}"
  mani="$(oras manifest fetch --plain-http -u "${MH_USER}" -p "${MH_PASS}" \
          "${MH_URL}/${LAYER_REPO}@${child}")"
  echo "${mani}" | jq .
fi
n_layers="$(echo "${mani}" | jq '.layers | length')"
mt="$(echo "${mani}" | jq -r '.layers[0].mediaType')"
echo "   layers=${n_layers} layer0.mediaType=${mt}"

echo "== assertions =="
[ "${n_layers}" = "2" ] || { echo "FAIL: expected 2 layers, got ${n_layers}"; exit 1; }
[ "${mt}" = "application/vnd.oci.image.layer.v1.tar" ] || {
  echo "FAIL: expected uncompressed tar layer, got ${mt}"; exit 1; }
echo "${mani}" | jq -e '.layers | length == 2 and all(.[]; .mediaType == "application/vnd.oci.image.layer.v1.tar")' >/dev/null \
  || { echo "FAIL: layer mediaTypes not all image.layer.v1.tar"; exit 1; }

echo "== 4. runtime registry + model API (resolution stack) =="
# Register a runtime, then assert it and the repackaged model surface over the API
# — the pieces the model card's resolved composition consumes.
"${ADMIN}" --config "${CFG}" runtime register \
  --name llama-server --image "${MH_URL}/apps/llama-server:1" \
  --formats gguf --config-contract qnib.model.config.v1
rts="$(curl -fsS "http://${MH_URL}/api/v1/runtimes")"
echo "   runtimes: ${rts}"
echo "${rts}" | jq -e '.runtimes | map(.name) | index("llama-server") != null' >/dev/null \
  || { echo "FAIL: GET /api/v1/runtimes did not list llama-server"; exit 1; }
model="$(curl -fsS "http://${MH_URL}/api/v1/models/${NAME}")"
echo "   model: ${model}"
echo "${model}" | jq -e --arg q "${QUANT}" '.model.quantizations | map(.tag) | index($q) != null' >/dev/null \
  || { echo "FAIL: GET /api/v1/models/${NAME} missing quant ${QUANT}"; exit 1; }
echo "   OK: runtime registered + model resolvable under aggregated profile ${MH_PROFILE}"

echo "== 5. deploy path: model injected into a runtime image pulled under the profile =="
# The payoff of repackage: a plain runtime image pulled under the model's
# aggregated profile comes back with the model-layer injected at /model/<name>/.
# This exercises the pull-time resolution + manipulation engine end-to-end and
# guards the two injection fixes (quant-rule platform gate; manipulated-manifest
# platform hydration) — no GPU or real inference runtime needed, a stand-in base
# image suffices to prove the bytes are injected and the tag pull is resolvable.
RUNTIME_IMG="${MH_URL}/apps/${NAME}-runtime:1"
docker pull -q alpine:3.19 >/dev/null
docker tag alpine:3.19 "${RUNTIME_IMG}"
# The pull/push identity encodes the intent: user 'oci', profile ${MH_PROFILE}
# (the '#' in the aggregated profile is fine in a docker-login username).
docker login "${MH_URL}" -u "${MH_USER}/${MH_PROFILE}" -p "${MH_PASS}" >/dev/null 2>&1
docker push "${RUNTIME_IMG}" >/dev/null       # push under the profile → compute-time injection
docker rmi "${RUNTIME_IMG}" alpine:3.19 >/dev/null 2>&1 || true
docker pull -q "${RUNTIME_IMG}" >/dev/null    # pull under the profile → the injected image
injected="$(docker run --rm --entrypoint sh "${RUNTIME_IMG}" -c "ls /model/${NAME}/ 2>/dev/null | wc -l" | tr -d '[:space:]')"
[ "${injected:-0}" -ge 1 ] || {
  echo "FAIL: no model files injected at /model/${NAME}/ when pulling a runtime under ${MH_PROFILE}"; exit 1; }
echo "   OK: ${injected} model file(s) injected at /model/${NAME}/ via pull under ${MH_PROFILE}"

echo "PASS: repackaged + runtime/model API OK + model injected into a runtime pulled under ${MH_PROFILE}"
