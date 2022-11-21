#!/bin/bash

TOPDIR=$(git rev-parse --show-toplevel)
go build -o ${TOPDIR}/metahub-admin