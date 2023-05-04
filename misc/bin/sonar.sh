#!/bin/bash
set -x 

sonar-scanner \
  -Dsonar.branch.name=$(git rev-parse --abbrev-ref HEAD) \
  -Dsonar.projectKey=${SONAR_QUBE_PROJECT} \
  -Dsonar.sources=. \
  -Dsonar.host.url=${SONAR_QUBE_URL} \
  -Dsonar.login=${SONAR_QUBE_TOKEN}
