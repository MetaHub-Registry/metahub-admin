#!/bin/bash
set -e

if [[ -f /run/secrets/aws-access-key ]];then
    export AWS_ACCESS_KEY_ID=$(cat /run/secrets/aws-access-key)
fi
if [[ -f /run/secrets/aws-access-key ]];then
    export AWS_SECRET_ACCESS_KEY=$(cat /run/secrets/aws-secret-access-key)
fi

aws ecr get-login-password --region eu-west-1 | docker login --username AWS --password-stdin 193198941246.dkr.ecr.eu-west-1.amazonaws.com

docker-compose -f docker-compose-cicd.yaml pull
