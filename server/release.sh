#!/bin/bash

BIN_NAME=WindSend-Relay
GO_PACKAGE_NAME=github.com/doraemonkeys/WindSend-Relay
ZIP_NAME_PREFIX=WindSend-Relay
BIN_DIR=$(
    mkdir -p dist
    echo 'dist'
)
BRANCH=$(git branch --show-current)
COMMIT_SHORT_HASH=$(git rev-parse --short HEAD)
COMMIT_HASH=$(git rev-parse HEAD)
BUILD_TIME=$(date -Iseconds --utc)
# BUILD_TAG=$(git describe --tags --abbrev=0 || echo 'unknown')
BUILD_TAG=$(git tag --sort=-creatordate | head -n 1 || echo 'unknown')

ZIP_NAME=${BRANCH}-${BUILD_TAG}-${COMMIT_SHORT_HASH}
if [ "$BRANCH" = "main" ] || [ "$BRANCH" = "master" ]; then
    ZIP_NAME=${BUILD_TAG}-${COMMIT_SHORT_HASH}
fi

GOBUILDCMD=(go build -trimpath -ldflags "-X '${GO_PACKAGE_NAME}/version.BuildHash=${COMMIT_HASH}' \
										 -X '${GO_PACKAGE_NAME}/version.BuildTime=${BUILD_TIME}' \
										 -w -s")

function build_windows_amd64() {
    GOOS=windows GOARCH=amd64 GOAMD64=v3 "${GOBUILDCMD[@]}" -o "${BIN_DIR}"/${ZIP_NAME_PREFIX}-windows-x64-"${ZIP_NAME}"/${BIN_NAME}.exe
}

function build_windows_amd64_compatible() {
    GOOS=windows GOARCH=amd64 GOAMD64=v1 "${GOBUILDCMD[@]}" -o "${BIN_DIR}"/${ZIP_NAME_PREFIX}-windows-x64-compatible-"${ZIP_NAME}"/${BIN_NAME}.exe
}

function build_windows_arm64() {
    GOOS=windows GOARCH=arm64 "${GOBUILDCMD[@]}" -o "${BIN_DIR}"/${ZIP_NAME_PREFIX}-windows-arm64-"${ZIP_NAME}"/${BIN_NAME}.exe
}

function build_windows_arm32v7() {
    GOOS=windows GOARCH=arm GOARM=7 "${GOBUILDCMD[@]}" -o "${BIN_DIR}"/${ZIP_NAME_PREFIX}-windows-arm32v7-"${ZIP_NAME}"/${BIN_NAME}.exe
}

function build_linux_amd64() {
    GOOS=linux GOARCH=amd64 GOAMD64=v3 "${GOBUILDCMD[@]}" -o "${BIN_DIR}"/${ZIP_NAME_PREFIX}-linux-amd64-"${ZIP_NAME}"/${BIN_NAME}
}

function build_linux_amd64_compatible() {
    GOOS=linux GOARCH=amd64 GOAMD64=v1 "${GOBUILDCMD[@]}" -o "${BIN_DIR}"/${ZIP_NAME_PREFIX}-linux-amd64-compatible-"${ZIP_NAME}"/${BIN_NAME}
}

function build_linux_arm64() {
    GOOS=linux GOARCH=arm64 "${GOBUILDCMD[@]}" -o "${BIN_DIR}"/${ZIP_NAME_PREFIX}-linux-arm64-"${ZIP_NAME}"/${BIN_NAME}
}

function build_linux_armv5() {
    GOOS=linux GOARCH=arm GOARM=5 "${GOBUILDCMD[@]}" -o "${BIN_DIR}"/${ZIP_NAME_PREFIX}-linux-armv5-"${ZIP_NAME}"/${BIN_NAME}
}

function build_linux_armv6() {
    GOOS=linux GOARCH=arm GOARM=6 "${GOBUILDCMD[@]}" -o "${BIN_DIR}"/${ZIP_NAME_PREFIX}-linux-armv6-"${ZIP_NAME}"/${BIN_NAME}
}

function build_linux_armv7() {
    GOOS=linux GOARCH=arm GOARM=7 "${GOBUILDCMD[@]}" -o "${BIN_DIR}"/${ZIP_NAME_PREFIX}-linux-armv7-"${ZIP_NAME}"/${BIN_NAME}
}

function build_darwin_amd64() {
    GOOS=darwin GOARCH=amd64 GOAMD64=v3 "${GOBUILDCMD[@]}" -o "${BIN_DIR}"/${ZIP_NAME_PREFIX}-darwin-amd64-"${ZIP_NAME}"/${BIN_NAME}
}

function build_darwin_amd64_compatible() {
    GOOS=darwin GOARCH=amd64 GOAMD64=v1 "${GOBUILDCMD[@]}" -o "${BIN_DIR}"/${ZIP_NAME_PREFIX}-darwin-amd64-compatible-"${ZIP_NAME}"/${BIN_NAME}
}

function build_darwin_arm64() {
    GOOS=darwin GOARCH=arm64 "${GOBUILDCMD[@]}" -o "${BIN_DIR}"/${ZIP_NAME_PREFIX}-darwin-arm64-"${ZIP_NAME}"/${BIN_NAME}
}

function gz_release() {
    cd "${BIN_DIR}" || exit
    find . -type d -not -name "${ZIP_NAME_PREFIX}-windows-*" -not -name '.' -not -name '..' -exec tar -zcf {}.tar.gz {} \;
    cd - || exit
}

function zip_release() {
    cd "${BIN_DIR}" || exit
    find . -type d -name "${ZIP_NAME_PREFIX}-windows-*" -not -name '.' -not -name '..' -exec zip -r {}.zip {} \;
    cd - || exit
}

cleanup() {
    echo "Cleaning up old builds..."
    rm -rf "${BIN_DIR}"
    mkdir -p "${BIN_DIR}"
    rm -rf version.txt
    rm -rf release.md
    rm -rf release.txt

    echo "Cleanup complete."
}

main() {
    echo "COMMITHASH: $COMMIT_HASH"
    echo "BINDIR: $BIN_DIR"
    echo "BRANCH: $BRANCH"
    echo "BUILD_TIME: $BUILD_TIME"
    echo "BUILD_TAG: $BUILD_TAG"

    case $1 in
    "all")
        build_windows_amd64
        build_windows_amd64_compatible
        build_windows_arm64
        build_windows_arm32v7
        build_linux_amd64
        build_linux_amd64_compatible
        build_linux_arm64
        build_linux_armv5
        build_linux_armv6
        build_linux_armv7
        build_darwin_amd64
        build_darwin_amd64_compatible
        build_darwin_arm64
        ;;
    "linux")
        build_linux_amd64
        build_linux_amd64_compatible
        build_linux_arm64
        build_linux_armv5
        build_linux_armv6
        build_linux_armv7
        ;;
    "macos")
        build_darwin_amd64
        build_darwin_amd64_compatible
        build_darwin_arm64
        ;;
    "windows")
        build_windows_amd64
        build_windows_amd64_compatible
        build_windows_arm64
        build_windows_arm32v7
        ;;
    "clean")
        cleanup
        exit 0
        ;;
    *)
        echo "Usage: $0 {all|linux|macos|windows|clean}"
        exit 1
        ;;
    esac

    {
        echo "BRANCH: $BRANCH"
        echo "COMMITHASH: $COMMIT_HASH"
        echo "BUILD_TAG: $BUILD_TAG"
        echo "BUILD_TIME: $BUILD_TIME"
    } >"${BIN_DIR}"/version.txt

    echo "Build complete."

    chmod -R +x "${BIN_DIR}"/*

    gz_release
    zip_release
}

main "$1"
