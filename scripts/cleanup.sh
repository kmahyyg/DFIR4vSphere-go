#!/usr/bin/env bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
cd ${SCRIPT_DIR}/../
rm -rf bin/*
mkdir -p bin/
rm -f pkg/common/gitversion.txt
. ${SCRIPT_DIR}/versioning.sh