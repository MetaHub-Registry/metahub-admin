# syntax=docker/dockerfile:1.4
FROM golang:1.19.1-alpine3.16 AS go
RUN apk add --no-cache git gcc musl-dev
WORKDIR /go/metahub-admin
COPY . ./
RUN --mount=type=cache,target=/go/pkg/mod go mod tidy
ENV CGO_ENABLED=1 GOOS=linux
RUN --mount=type=cache,target=/go/pkg/mod go build -a -ldflags '-extldflags "-static"' .

FROM alpine:3.16.2
COPY --from=go /go/metahub-admin/metahub-admin /usr/bin/metahub-admin
