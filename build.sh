#!/bin/bash

eval $(minikube docker-env)

DOCKER_ENV=$1
if [ -z "$DOCKER_ENV" ]; then
    DOCKER_ENV="dev"
fi

docker buildx build -t sharded-kvs-node -f docker/node/Dockerfile.$1 .
