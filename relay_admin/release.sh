#!/bin/bash


ZIP_NAME_PREFIX=WindSend-Relay-Admin-Frontend
BRANCH=$(git branch --show-current)
COMMIT_SHORT_HASH=$(git rev-parse --short HEAD)
BUILD_TAG=$(git tag --sort=-creatordate | head -n 1 || echo 'unknown')

ZIP_NAME=${BRANCH}-${BUILD_TAG}-${COMMIT_SHORT_HASH}
if [ "$BRANCH" = "main" ] || [ "$BRANCH" = "master" ]; then
    ZIP_NAME=${BUILD_TAG}-${COMMIT_SHORT_HASH}
fi

npm run build
zip -r "${ZIP_NAME_PREFIX}-${ZIP_NAME}.zip" ./dist
