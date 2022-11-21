#!/bin/bash


### Copy /minio to current dir and upload to artefacts
CNT_NAME=$(docker ps|awk '/stack.*backup/{print $NF}')
docker exec -t --workdir /minio ${CNT_NAME} tar cf /backup/minio.tar .
docker-compose -p admin-stack -f docker-compose-data.yaml down -v
