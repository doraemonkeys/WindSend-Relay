#!/bin/bash

COMMIT_HASH=$(git rev-parse HEAD)
BUILD_TIME=$(date -Iseconds --utc)
# BUILD_TAG=$(git describe --tags --abbrev=0 || echo 'unknown')
BUILD_TAG=$(git tag --sort=-creatordate | head -n 1 || echo 'unknown')
if [ "${BUILD_TAG}" = "" ]; then
    BUILD_TAG="v0.0.0"
fi

echo "BUILD_TAG: ${BUILD_TAG}"
echo "COMMIT_HASH: ${COMMIT_HASH}"
echo "BUILD_TIME: ${BUILD_TIME}"

docker build \
    --build-arg BUILD_HASH="${COMMIT_HASH}" \
    --build-arg BUILD_TIME="${BUILD_TIME}" \
    -t windsend-relay:"${BUILD_TAG}" .

# docker push doraemonkey/windsend-relay:"${BUILD_TAG}"
