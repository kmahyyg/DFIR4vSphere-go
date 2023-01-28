#!/bin/bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
cd ${SCRIPT_DIR}/../

if [[ -z "$1" ]]; then
  echo "Usage: ./macos_universal.sh filename_base"
  exit 2
fi

FILE_BASENAME="$1"

lipo -create -output "./bin/${FILE_BASENAME}-darwin-universal" "./bin/${FILE_BASENAME}-darwin-amd64" "./bin/${FILE_BASENAME}-darwin-arm64"
rm -f "./bin/${FILE_BASENAME}-darwin-arm64" "./bin/${FILE_BASENAME}-darwin-amd64"
