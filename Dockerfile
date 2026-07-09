# syntax=docker/dockerfile:1.4
ARG METALIB_IMAGE
FROM $METALIB_IMAGE AS metalib

FROM alpine:3.22 AS gonja
RUN apk add --no-cache wget \
 && wget -qO- https://gitlab.com/qnib-golang/gonja-cli/-/releases/v0.0.2/downloads/gonja-cli_0.0.2_linux_amd64.tar.gz |tar xfz - -C /usr/local/bin

FROM alpine:3.22 AS certs
RUN apk update && apk add ca-certificates

FROM golang:1.24.4 AS cbuilder
WORKDIR /go/metahub-admin
COPY . ./
COPY --from=metalib /pkg ext/metalib/pkg
COPY --from=metalib /go.sum /go.mod ext/metalib/
RUN go mod edit -replace github.com/metahub-registry/metahub-core=/go/metahub-admin/ext/metalib
RUN find ext/metalib
RUN --mount=type=cache,target=/go/pkg/mod go mod tidy
ENV CGO_ENABLED=1 GOOS=linux
RUN --mount=type=cache,target=/go/pkg/mod go build -a -ldflags '-extldflags "-static"' .

FROM busybox:1.35.0 AS busybox
COPY --from=cbuilder /go/metahub-admin/metahub-admin /usr/bin/metahub-admin
COPY --from=certs /etc/ssl/certs /etc/ssl/certs

FROM golang:1.24-alpine3.22 AS dev
RUN apk add --no-cache git gcc musl-dev bash vim curl wget
COPY ./misc/docker/dev/bash_history /root/.bash_history

FROM golang:1.24-alpine3.22 AS go
RUN apk add --no-cache git gcc musl-dev vim bash
WORKDIR /go/metahub-admin
COPY . ./
COPY --from=metalib /pkg ext/metalib/pkg
COPY --from=metalib /go.sum /go.mod ext/metalib/
RUN go mod edit -replace github.com/metahub-registry/metahub-core=/go/metahub-admin/ext/metalib
RUN --mount=type=cache,target=/go/pkg/mod go mod tidy
ENV CGO_ENABLED=1 GOOS=linux
RUN --mount=type=cache,target=/go/pkg/mod go build -a -ldflags '-extldflags "-static"' .

FROM alpine:3.22
RUN apk add ca-certificates bash git curl jq
# oras CLI: the product image doubles as the model-seed/repackage image (the
# gpu-workload-test Seed Job runs `oras cp` + `metahub-admin repackage` from it),
# so the pipeline-built admin image always carries a matching oras + admin binary.
ARG ORAS_VERSION=1.2.0
RUN curl -fsSL -o /tmp/oras.tgz \
      "https://github.com/oras-project/oras/releases/download/v${ORAS_VERSION}/oras_${ORAS_VERSION}_linux_amd64.tar.gz" \
 && tar xzf /tmp/oras.tgz -C /usr/local/bin oras \
 && rm -f /tmp/oras.tgz \
 && oras version
COPY --from=go /go/metahub-admin/metahub-admin /usr/bin/metahub-admin
COPY --from=gonja /usr/local/bin/gonja-cli /usr/local/bin/gonja-cli
COPY ./misc/bin/run.sh /usr/bin/run.sh
RUN chmod +x /usr/bin/run.sh
COPY ./misc/tpl/metahub-admin.yaml.tpl /etc//metahub/metahub-admin.yaml.tpl