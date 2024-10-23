#!/bin/bash

SCRIPT=${BASH_SOURCE[0]}
SCRIPT_DIR=$(cd -P -- "$(dirname "${SCRIPT}")" && pwd)
SRC_DIR=$(cd -P -- "${SCRIPT_DIR}/.." && pwd)

BUILD_DIR=/tmp/reaper-test.$$


#
#  Builds a docker reaper test image for the given fixtures.
#
function _build_image() {
  local image=${1:-"reaper/test"}
  local fixtures=${2:-"fixtures/no-config"}

  trap "rm -rf ${BUILD_DIR}; exit" 0 1 2 3 15

  echo "  - Building image ${image} ... "

  mkdir -p "${BUILD_DIR}"

  pushd "${SRC_DIR}" > /dev/null

  echo "  - building testpid1 ... "
  go build testpid1.go

  popd

  cp    "${SRC_DIR}/testpid1"    "${BUILD_DIR}"
  cp -r "${SRC_DIR}/bin"         "${BUILD_DIR}"
  cp -r "${fixtures}"/*          "${BUILD_DIR}"

  pushd "${BUILD_DIR}" > /dev/null
  echo "  - building ${image} ... "
  docker build -t "${image}" .
  popd

  docker images "${image}"

}  #  End of function  _build_image.


#
#  main():
#
_build_image "$@"
