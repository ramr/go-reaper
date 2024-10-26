#!/bin/bash

set -euo pipefail

SCRIPT=${BASH_SOURCE[0]}
readonly SCRIPT

SCRIPT_DIR=$(cd -P -- "$(dirname "${SCRIPT}")" && pwd)
readonly SCRIPT_DIR

TEST_SRC_DIR=$(cd -P -- "${SCRIPT_DIR}/.." && pwd)
readonly TEST_SRC_DIR


#
#  Builds a docker reaper test image for the given fixtures.
#
#  Usage:  _build_image  <image-name>  <fixtures>
#
#  Examples:
#      _build_image reaper/test fixtures/no-config
#
#      _build_image reaper/debug-on-test fixtures/json-config/debug-on
#
#      _build_image reaper/notify-test fixtures/json-config/notify
#
function _build_image() {
    local image=${1:-"reaper/test"}
    local fixtures=${2:-"fixtures/no-config"}

    cd "${SCRIPT_DIR}"

    trap 'rm -rf "${SCRIPT_DIR}/reaper-image" exit' 0 1 2 3 15

    echo "  - Building image ${image} ... "

    echo "  - Recreating reaper-image directory ..."
    rm -rf reaper-image
    mkdir -p reaper-image

    cd "${TEST_SRC_DIR}" || exit 70

    echo "  - Copying source and binaries to reaper-image ..."
    cp    "${TEST_SRC_DIR}/testpid1.go"  "${SCRIPT_DIR}/reaper-image"
    cp -r "${TEST_SRC_DIR}/bin"          "${SCRIPT_DIR}/reaper-image"

    echo "  - Copying fixtures to reaper-image ..."
    cp -r "${fixtures}"/*                "${SCRIPT_DIR}/reaper-image"

    cd "${SCRIPT_DIR}/reaper-image" || exit 70

    echo "  - Rebuilding testpid1 ... "
    rm -f testpid1
    go build testpid1.go

    echo "  - Building ${image} ... "
    docker build -t "${image}" .

    docker images "${image}"

}  #  End of function  _build_image.


#
#  main():
#
_build_image "$@"
