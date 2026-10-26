# syntax=docker/dockerfile:1.4
#
# Multi-arch build (linux/amd64 + linux/arm64). CI fans out: each arch is
# built on an arch-matched agent (k8s-dind-amd64 / k8s-dind-arm64) with plain
# `docker build --platform linux/<arch>` and pushed as :<TAG>-<arch>; a
# Manifest stage assembles the two into a manifest list. Consequences:
#   * Stages that EXECUTE commands are pinned to --platform=$BUILDPLATFORM.
#     In CI the build agent IS the target arch, so everything runs natively —
#     no QEMU anywhere. (The pin also keeps ad-hoc one-shot buildx builds
#     working on a foreign host, with zig cc covering the Go cross-compile.)
#   * RUN --mount=type=cache is used for GOMODCACHE: modules downloaded onto
#     an overlayfs rootfs break on `.wh.*` names (regclient testdata) —
#     overlayfs reserves the prefix as whiteout markers. The cache backend
#     is a plain volume.
#   * The final stage stays unpinned (alpine is multi-arch); its RUNs (apk,
#     oras fetch with a TARGETARCH-aware URL, chmod) execute natively in CI.
ARG METALIB_IMAGE
FROM --platform=$BUILDPLATFORM $METALIB_IMAGE AS metalib

# gonja-cli is fetched for the TARGET arch; the fetch itself runs natively.
FROM --platform=$BUILDPLATFORM mirror.gcr.io/library/alpine:3.22 AS gonja
ARG TARGETARCH
RUN apk add --no-cache wget \
 && wget -qO- https://gitlab.com/qnib-golang/gonja-cli/-/releases/v0.0.2/downloads/gonja-cli_0.0.2_linux_${TARGETARCH}.tar.gz |tar xfz - -C /usr/local/bin

# CA bundle is arch-independent; the stage is pinned so apk runs natively.
FROM --platform=$BUILDPLATFORM mirror.gcr.io/library/alpine:3.22 AS certs
RUN apk update && apk add ca-certificates

# --- Go build: native on the builder, cross-compiles to TARGETARCH ---------
# Alpine base (golang:1.26 is Debian — no apk): zig comes from the alpine
# package repo and serves as the cgo cross-compiler.
FROM --platform=$BUILDPLATFORM mirror.gcr.io/library/golang:1.26-alpine AS cbuilder
ARG TARGETOS TARGETARCH
RUN apk add --no-cache git zig
WORKDIR /go/metahub-admin
COPY . ./
COPY --from=metalib /pkg ext/metahub-core/pkg
COPY --from=metalib /go.sum /go.mod ext/metahub-core/
RUN go mod edit -replace github.com/metahub-registry/metahub-core=/go/metahub-admin/ext/metahub-core
RUN find ext/metahub-core
RUN --mount=type=cache,target=/go/pkg/mod go mod tidy
ENV CGO_ENABLED=1 GOOS=linux
# -fno-sanitize=undefined: zig cc instruments C with UBSan by default but does
# not link the ubsan runtime as a pure linker (undefined __ubsan_handle_*).
ENV CGO_CFLAGS="-fno-sanitize=undefined"
# The build itself reuses the GOMODCACHE cache mount: modules downloaded onto
# the overlayfs rootfs break when a dependency's module zip carries `.wh.*`
# testdata files (regclient v0.4.5) — overlayfs reserves the `.wh.` prefix as
# whiteout markers and rejects them with EINVAL ("invalid argument"). The
# cache-mount backend (local volume) has no such restriction.
RUN --mount=type=cache,target=/go/pkg/mod case "${TARGETARCH}" in \
      amd64) ZTARGET="x86_64-linux-musl" ;; \
      arm64) ZTARGET="aarch64-linux-musl" ;; \
      *) echo "unsupported TARGETARCH: ${TARGETARCH}" >&2; exit 1 ;; \
    esac \
 && CC="zig cc -target ${ZTARGET}" \
    CGO_ENABLED=1 GOOS="${TARGETOS:-linux}" GOARCH="${TARGETARCH}" \
    go build -a -ldflags '-extldflags "-static"' .

FROM mirror.gcr.io/library/busybox:1.35.0 AS busybox
COPY --from=cbuilder /go/metahub-admin/metahub-admin /usr/bin/metahub-admin
COPY --from=certs /etc/ssl/certs /etc/ssl/certs

FROM --platform=$BUILDPLATFORM mirror.gcr.io/library/golang:1.26-alpine AS dev
RUN apk add --no-cache git gcc musl-dev bash vim curl wget
COPY ./misc/docker/dev/bash_history /root/.bash_history

# --- Go build (second target of the same source) ---------------------------
FROM --platform=$BUILDPLATFORM mirror.gcr.io/library/golang:1.26-alpine AS go
ARG TARGETOS TARGETARCH
RUN apk add --no-cache git gcc musl-dev vim bash zig
WORKDIR /go/metahub-admin
COPY . ./
COPY --from=metalib /pkg ext/metahub-core/pkg
COPY --from=metalib /go.sum /go.mod ext/metahub-core/
RUN go mod edit -replace github.com/metahub-registry/metahub-core=/go/metahub-admin/ext/metahub-core
RUN --mount=type=cache,target=/go/pkg/mod go mod tidy
ENV CGO_ENABLED=1 GOOS=linux
# -fno-sanitize=undefined: see note in the cbuilder stage
ENV CGO_CFLAGS="-fno-sanitize=undefined"
RUN --mount=type=cache,target=/go/pkg/mod case "${TARGETARCH}" in \
      amd64) ZTARGET="x86_64-linux-musl" ;; \
      arm64) ZTARGET="aarch64-linux-musl" ;; \
      *) echo "unsupported TARGETARCH: ${TARGETARCH}" >&2; exit 1 ;; \
    esac \
 && CC="zig cc -target ${ZTARGET}" \
    CGO_ENABLED=1 GOOS="${TARGETOS:-linux}" GOARCH="${TARGETARCH}" \
    go build -a -ldflags '-extldflags "-static"' .

# --- runtime (per-arch; RUNs run under QEMU for the foreign leg) -----------
FROM mirror.gcr.io/library/alpine:3.22
ARG TARGETARCH
RUN apk add ca-certificates bash git curl jq
# oras CLI: the product image doubles as the model-seed/repackage image (the
# gpu-workload-test Seed Job runs `oras cp` + `metahub-admin repackage` from it),
# so the pipeline-built admin image always carries a matching oras + admin binary.
ARG ORAS_VERSION=1.2.0
RUN curl -fsSL -o /tmp/oras.tgz \
      "https://github.com/oras-project/oras/releases/download/v${ORAS_VERSION}/oras_${ORAS_VERSION}_linux_${TARGETARCH}.tar.gz" \
 && tar xzf /tmp/oras.tgz -C /usr/local/bin oras \
 && rm -f /tmp/oras.tgz \
 && oras version
COPY --from=go /go/metahub-admin/metahub-admin /usr/bin/metahub-admin
COPY --from=gonja /usr/local/bin/gonja-cli /usr/local/bin/gonja-cli
COPY ./misc/bin/run.sh /usr/bin/run.sh
RUN chmod +x /usr/bin/run.sh
COPY ./misc/tpl/metahub-admin.yaml.tpl /etc//metahub/metahub-admin.yaml.tpl
