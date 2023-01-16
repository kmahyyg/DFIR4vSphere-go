#!/bin/bash

export PROG_VERSION=$(git describe --long --tags --always --dirty)
# do not include any line break characters, use -n, they are illegal in http headers
echo -n "${PROG_VERSION}" > ./pkg/common/gitversion.txt
