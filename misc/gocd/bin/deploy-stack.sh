#!/bin/bash

function getMinioPort() {
    CNT_NAME=$(docker ps|awk '/stack.*minio/{print $NF}')
    MINIO_PORT=$(docker inspect ${CNT_NAME} |jq -r '.[] |.NetworkSettings.Ports."9000/tcp"[]|.HostPort')
    N=0
    if [[ -z ${MINIO_PORT} ]];then
        if [[ ${N} -gt 10 ]];then
            echo "ERROR: Could not find Minio port"
            exit 1
        fi
        N=$((N+1))
        sleep 1
        MINIO_PORT=$(docker inspect ${CNT_NAME} |jq -r '.[] |.NetworkSettings.Ports."9000/tcp"[]|.HostPort')   
    fi
    echo ${MINIO_PORT}
}

function getNeo4jBoltPort() {
    CNT_NAME=$(docker ps|awk '/stack.*neo4j/{print $NF}')
    NEO_PORT=$(docker inspect ${CNT_NAME} |jq -r '.[] |.NetworkSettings.Ports."7687/tcp"[].HostPort')
    N=0
    if [[ -z ${NEO_PORT} ]];then
        if [[ ${N} -gt 10 ]];then
            echo "ERROR: Could not find Neo port"
            exit 1
        fi
        N=$((N+1))
        sleep 1
        NEO_PORT=$(docker inspect ${CNT_NAME} |jq -r '.[] |.NetworkSettings.Ports."7687/tcp"[].HostPort')   
    fi
    echo ${NEO_PORT}
}

function getNeo4jHttpPort() {
    CNT_NAME=$(docker ps|awk '/stack.*neo4j/{print $NF}')
    NEO_HTTP_PORT=$(docker inspect ${CNT_NAME} |jq -r '.[] |.NetworkSettings.Ports."7474/tcp"[].HostPort')
    N=0
    if [[ -z ${NEO_HTTP_PORT} ]];then
        if [[ ${N} -gt 10 ]];then
            echo "ERROR: Could not find Neo port"
            exit 1
        fi
        N=$((N+1))
        sleep 1
        NEO_HTTP_PORT=$(docker inspect ${CNT_NAME} |jq -r '.[] |.NetworkSettings.Ports."7474/tcp"[].HostPort')   
    fi
    echo ${NEO_HTTP_PORT}
}


function checkMinioHealth() {
    MINIO_PORT=$(getMinioPort)
    N=0
    curl -s http://localhost:${MINIO_PORT}/minio/health/live
    echo -n "Check Minio health endpoint: "

    while [[ $? -ne 0 ]];do
        if [[ ${N} -gt 10 ]];then
            echo "ERROR: Could not find Neo port"
            exit 1
        fi
        N=$((N+1))
        sleep 1
        curl -s http://localhost:${MINIO_PORT}/minio/health/live
        echo -n "."
    done
    echo ""
    echo "-> Minio is healthy"
}

function checkNeo4jHealth() {
    NEO_HTTP_PORT=$(getNeo4jHttpPort)
    N=0
    curl -s http://localhost:${NEO_HTTP_PORT} >>/dev/null
    echo -n "Check Neo4j HTTP health endpoint: "
    while [[ $? -ne 0 ]];do
        if [[ ${N} -gt 10 ]];then
            echo "ERROR: Could not find Neo port"
            exit 1
        fi
        N=$((N+1))
        sleep 1
        curl -s http://localhost:${NEO_HTTP_PORT} >>/dev/null
        echo -n "."
    done
    echo ""
    echo "-> Neo4j is healthy"
}


docker-compose -p admin-stack -f docker-compose-data.yaml up -d
MINIO_PORT=$(getMinioPort)
NEO_HTTP_PORT=$(getNeo4jHttpPort)
NEO_BOLT_PORT=$(getNeo4jBoltPort)

checkMinioHealth
checkNeo4jHealth