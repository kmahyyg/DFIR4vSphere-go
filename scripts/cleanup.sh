#!/usr/bin/env bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
cd ${SCRIPT_DIR}/../
rm -rf bin/*
mkdir -p bin/
rm -f pkg/common/gitversion.txt
export PROG_VERSION=$(git describe --long --tags --always --dirty)
# do not include any line break characters, use -n, they are illegal in http headers
echo -n "${PROG_VERSION}" > ./pkg/common/gitversion.txt
