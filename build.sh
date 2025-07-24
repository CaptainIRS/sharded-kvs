#!/bin/bash

DOCKER_ENV=$1
if [ -z "$DOCKER_ENV" ]; then
    DOCKER_ENV="dev"
fi

docker buildx build -t sharded-kvs-shard -f docker/shard/Dockerfile.$1 .

kind load docker-image sharded-kvs-shard --name sharded-kvs-cluster
