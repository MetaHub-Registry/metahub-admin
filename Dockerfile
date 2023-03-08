# syntax=docker/dockerfile:1.4
FROM alpine as certs
RUN apk update && apk add ca-certificates

FROM golang:1.19.1 AS cbuilder
WORKDIR /go/metahub-admin
COPY . ./
RUN --mount=type=cache,target=/go/pkg/mod go mod tidy
ENV CGO_ENABLED=1 GOOS=linux
RUN --mount=type=cache,target=/go/pkg/mod go build -a -ldflags '-extldflags "-static"' .

FROM busybox:1.35.0 AS busybox
COPY --from=cbuilder /go/metahub-admin/metahub-admin /usr/bin/metahub-admin
COPY --from=certs /etc/ssl/certs /etc/ssl/certs

FROM golang:1.19.1-alpine3.16 AS dev
RUN apk add --no-cache git gcc musl-dev bash vim curl wget
COPY ./misc/docker/dev/bash_history /root/.bash_history

FROM golang:1.19.1-alpine3.16 AS go
RUN apk add --no-cache git gcc musl-dev
WORKDIR /go/metahub-admin
COPY . ./
RUN --mount=type=cache,target=/go/pkg/mod go mod tidy
ENV CGO_ENABLED=1 GOOS=linux
RUN --mount=type=cache,target=/go/pkg/mod go build -a -ldflags '-extldflags "-static"' .

FROM alpine:3.16.2
COPY --from=go /go/metahub-admin/metahub-admin /usr/bin/metahub-admin