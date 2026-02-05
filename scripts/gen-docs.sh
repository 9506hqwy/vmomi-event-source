#!/bin/bash
set -uo pipefail

SCRIPTS_PATH=$(dirname "$(readlink -f "$0")")
BASE_PATH=$(dirname "${SCRIPTS_PATH}")
MODULE_NAME=$(basename "${BASE_PATH}")
SERVER='localhost:6060'

go tool godoc -http="${SERVER}" &
curl --retry-connrefused --retry 5 -m 10 -s -o /dev/null "http://${SERVER}/pkg"

pushd /tmp || exit
    wget \
        --quiet \
        --adjust-extension \
        --recursive \
        --convert-links \
        --page-requisites \
        --no-parent \
        "http://${SERVER}/pkg/github.com/9506hqwy/${MODULE_NAME}/"

    mkdir -p "${BASE_PATH}/artifacts"
    tar -zcf "${BASE_PATH}/artifacts/docs.tar.gz" -C "${SERVER}" .
popd || exit

kill %1
