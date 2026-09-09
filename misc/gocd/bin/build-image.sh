#!/bin/bash
# Build + push ONE ARCH of the metahub-admin PRODUCT image to Harbor.
#
# The pipeline fans out: two parallel Image jobs (ARCH=amd64 / ARCH=arm64),
# each pinned to an arch-matched agent (k8s-dind-amd64 / k8s-dind-arm64), push
# ${NAME}:${TAG}-${ARCH}; a downstream Manifest stage assembles them into a
# multi-arch manifest list via `docker manifest create`. This isolates
# per-arch failures, parallelizes the legs, and runs everything natively —
# no QEMU/binfmt, no buildx sidecar.
#
# This image doubles as the model-seed/repackage image the gpu-workload-test
# Seed Job runs, so a per-branch, always-current admin binary (linked against the
# branch's metalib) reaches the seed with no hand-baked metahub-admin-seed:1.
#
# Mirrors metahub-api's build-api.sh, with two deliberate differences:
#   * push target is HARBOR (metahub/admin/<group>/<pipeline>), not Zot — the
#     in-cluster Seed Job pulls it via the existing harbor-pull-secret; Zot is
#     in-cluster plain-http and not a trusted pull source for the Job.
#   * builds the final image target (admin + oras), not just the cbuilder binary.
set -e

if [[ -z "${ARCH}" ]]; then
   echo "ERROR: ARCH required (amd64|arm64) — the pipeline sets it per job"
   exit 1
fi
if [[ "${ARCH}" != "amd64" && "${ARCH}" != "arm64" ]]; then
   echo "ERROR: unsupported ARCH '${ARCH}' (amd64|arm64)"
   exit 1
fi

if [[ -f ~/gocd_func.sh ]]; then
   FUNC_SRC=~/gocd_func.sh
elif [[ -f /opt/qnib/gocd/src/gocd_func.sh ]]; then
   FUNC_SRC=/opt/qnib/gocd/src/gocd_func.sh
else
   echo "ERROR: No gocd_func.sh found"
   exit 1
fi
source ${FUNC_SRC}
dumpEnv

# strip any local `replace` directive (CI wires metalib via --build-arg)
cat go.mod | awk '!/^replace/' > go.mod.tmp
mv go.mod.tmp go.mod

# metalib base image (FROM $METALIB_IMAGE) from the upstream metalib pipeline.
if [[ -f metalib_result/result.yaml ]]; then
   echo ">> found metalib_result/result.yaml"
   cat metalib_result/result.yaml
   METALIB_IMAGE=$(yq -r .image metalib_result/result.yaml)
   cat >> result.yaml << EOF
metalib:
   image: ${METALIB_IMAGE}
EOF
else
   echo "ERROR: no metalib_result/result.yaml — need METALIB_IMAGE for the build"
   exit 1
fi

if [[ -z "${HARBOR_REG}" || -z "${HARBOR_TOKEN}" ]]; then
   echo "ERROR: HARBOR_REG/HARBOR_TOKEN required to push the admin image"
   exit 1
fi

NEW_IMAGE_NAME=$(echo ${HARBOR_REG}/metahub/admin/${GO_PIPELINE_GROUP_NAME}/${GO_PIPELINE_NAME} | tr '[:upper:]' '[:lower:]')
TAG="$(echo ${GO_PIPELINE_LABEL} | tr '[:upper:]' '[:lower:]')"
# Per-arch tag: the multi-arch manifest is assembled downstream (manifest-image.sh).
NEW_IMAGE=$(echo ${NEW_IMAGE_NAME}:${TAG}-${ARCH} | tr '[:upper:]' '[:lower:]')
echo "> NEW_IMAGE: ${NEW_IMAGE}"

echo "> Login to ${HARBOR_REG} as 'robot\$good'"
docker login ${HARBOR_REG} -u 'robot$good' -p "${HARBOR_TOKEN}"

set -x
# --- single-arch build on the arch-pinned agent ------------------------------
# Plain `docker build` is enough here: the job runs on an arch-matched agent
# (k8s-dind-amd64 / k8s-dind-arm64), so the build AND the final-stage RUNs
# (apk/oras/chmod) execute natively — no QEMU/binfmt, no buildx sidecar.
# The default (docker) buildkit driver still honors the Dockerfile's
# RUN --mount=type=cache mounts (the regclient .wh. overlayfs workaround)
# and emits a plain single-platform manifest — no attestation index — which
# is exactly what `docker manifest create` needs as a source downstream.
# The push goes through dockerd, which has harbor in INSECURE_REGISTRIES
# (plain HTTP), so no buildkitd registry config is needed either.
export DOCKER_BUILDKIT=1
docker build --platform linux/${ARCH} \
   --build-arg=METALIB_IMAGE=${METALIB_IMAGE} \
   -t ${NEW_IMAGE} .
docker push ${NEW_IMAGE}
# .full is the per-arch image; the Manifest stage publishes the manifest-list
# ref (no suffix) that downstream consumers (gpu-workload-test) run.
gdh patch -f result.yaml \
   --add=metahub.image.reg=${HARBOR_REG} \
   --add=metahub.image.name=${NEW_IMAGE_NAME} \
   --add=metahub.image.tag=${TAG} \
   --add=metahub.image.arch=${ARCH} \
   --add=metahub.image.full=${NEW_IMAGE}
