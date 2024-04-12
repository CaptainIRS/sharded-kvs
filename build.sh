#!/bin/bash

eval $(minikube docker-env)

docker buildx build -t sharded-kvs-node .
