#!/usr/bin/env bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
cd ${SCRIPT_DIR}/../
. ${SCRIPT_DIR}/cleanup.sh
. ${SCRIPT_DIR}/versioning.sh

export CGO_ENABLED=0
export BUILD_GCFLAG="all=-N\ -l"

export DBG_ADDITIONAL_LDFLAG=""
export REL_ADDITIONAL_LDFLAG="-s -w"

export GOC="go"

export REL_GOFLAG="-trimpath"

export BUILD_LDFLAG=""

go mod download
go mod tidy
go install mvdan.cc/garble@latest

if [[ -z $1 ]] || [[ -z $2 ]]; then
  echo "Usage: build.sh output-folder/executable-basename package-name"
  exit 1
fi

COMPILED_EXE_BASENAME=$1
PKG_TOBUILD=$2

SCRIPT_NAME=$(basename "$0")
FAILURES=""
PLATFORMS="windows/amd64 windows/arm64 linux/arm64 linux/amd64 linux/386 darwin/amd64 darwin/arm64"

if [[ ${BUILD_ENV} = "release" ]]; then
  for PLATFORM in $PLATFORMS; do
    GOOS=${PLATFORM%/*}
    GOARCH=${PLATFORM#*/}
    BIN_FILENAME="${COMPILED_EXE_BASENAME}-${GOOS}-${GOARCH}"
    if [[ "${GOOS}" == "windows" ]]; then BIN_FILENAME="${BIN_FILENAME}.exe"; fi
    CMD="CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} ${GOC} build ${REL_GOFLAG} -ldflags \"${BUILD_LDFLAG} ${REL_ADDITIONAL_LDFLAG}\" -o ${BIN_FILENAME} ${PKG_TOBUILD}"
    echo "${CMD}"
    eval "${CMD}" || FAILURES="${FAILURES} ${PLATFORM}"
    if [[ "${GOOS}" != "darwin" ]]; then
  	  strip "${BIN_FILENAME}" || true
    fi
  done
else
  CMD="CGO_ENABLED=0 ${GOC} build -gcflags ${BUILD_GCFLAG} -ldflags \"${BUILD_LDFLAG} ${DBG_ADDITIONAL_LDFLAG}\" -o ${COMPILED_EXE_BASENAME} ${PKG_TOBUILD}"
  echo "${CMD}"
  eval "${CMD}"
  exit 0
fi

if [[ "${FAILURES}" != "" ]]; then
  echo ""
  echo "${SCRIPT_NAME} failed on: ${FAILURES}"
  exit 1
fi