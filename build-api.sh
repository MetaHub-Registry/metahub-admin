#!/bin/bash

docker-compose -f docker-compose.yaml -f docker-compose-data.yaml up -d

docker-compose -f docker-compose.yaml -f docker-compose-data.yaml down