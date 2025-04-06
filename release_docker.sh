#!/bin/bash

COMMIT_HASH=$(git rev-parse HEAD)
BUILD_TIME=$(date -Iseconds --utc)
# BUILD_TAG=$(git describe --tags --abbrev=0 || echo 'unknown')
BUILD_TAG=$(git tag --sort=-creatordate | head -n 1 || echo 'unknown')

docker build \
    --build-arg BUILD_HASH="${COMMIT_HASH}" \
    --build-arg BUILD_TIME="${BUILD_TIME}" \
    -t windsend-relay:"${BUILD_TAG}" .

# docker push doraemonkey/windsend-relay:"${BUILD_TAG}"
