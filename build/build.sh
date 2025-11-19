#!/bin/bash
# Copyright (c) Huawei Technologies Co., Ltd. 2025. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

BASE_DIR=$(cd "$(dirname "$0")"; pwd)
PROJECT_DIR=$(cd "$(dirname "$0")"/..; pwd)
OUTPUT_DIR="${BASE_DIR}/../output/yuanrong/pattern/pattern_faas"
TAR_OUT_DIR="${PROJECT_DIR}/build/_output"
TAR_OUT_FILE="faasfunctions.tar.gz"
EXECUTOR_DIR="${PROJECT_DIR}/build/faas/executor-meta"
TEST_CERT_PATH="${GOROOT}/src/net/http/internal/testcert.go"
BUILD_TAG_FUNCTION="function"
echo LD_LIBRARY_PATH=$LD_LIBRARY_PATH
MODULE_NAME_LIST=("faasfrontend")

# go module prepare
export GO111MODULE=on
export GONOSUMDB=*
export CGO_ENABLED=1

# resolve missing go.sum entry
go env -w "GOFLAGS"="-mod=mod"

# remove hard coded cert file in net/http
[ -f "${TEST_CERT_PATH}" ] && rm -f "${TEST_CERT_PATH}"

# clean build history
bash "${BASE_DIR}"/clean.sh

cd "${PROJECT_DIR}"
. "${BASE_DIR}"/compile_functions.sh

# zip function file
FLAGS='-extldflags "-fPIC -fstack-protector-strong -Wl,-z,now,-z,relro,-z,noexecstack,-s -Wall -Werror"'
export CGO_CFLAGS="$CGO_CFLAGS -fstack-protector-strong -D_FORTIFY_SOURCE=2 -O2"
MODULE_NAME="faasfrontend"
cd "${TAR_OUT_DIR}"
mkdir -p "${MODULE_NAME}"
SO_PATH="${TAR_OUT_DIR}/${MODULE_NAME}/${MODULE_NAME}.so"
BIN_PATH="${TAR_OUT_DIR}/${MODULE_NAME}/${MODULE_NAME}"

CC='gcc -fstack-protector-strong -D_FORTIFY_SOURCE=2 -O2'
go build -tags "module" -buildmode=pie -ldflags "${FLAGS}" \
-o ${BIN_PATH} "${PROJECT_DIR}/cmd/${MODULE_NAME}/module_main.go"

CC='gcc -fstack-protector-strong -D_FORTIFY_SOURCE=2 -O2'
go build -tags "${BUILD_TAG_FUNCTION}" -buildmode=plugin -ldflags "${FLAGS}" \
-o ${SO_PATH} "${PROJECT_DIR}/cmd/${MODULE_NAME}/function_main.go"

chmod -R 500 ${SO_PATH}
cd "${MODULE_NAME}"
zip -r "${MODULE_NAME}.zip" *

cp "${PROJECT_DIR}/build/function_meta.json" "${TAR_OUT_DIR}/${MODULE_NAME}/${MODULE_NAME}_meta.json"

sed -i "s/moduleName/${MODULE_NAME}/g" "${TAR_OUT_DIR}/${MODULE_NAME}/${MODULE_NAME}_meta.json"

code_size_line=11
code_size_old=0
code_size_new=$(stat --format=%s "${TAR_OUT_DIR}/${MODULE_NAME}/${MODULE_NAME}.zip")
sed -i "${code_size_line} s@${code_size_old}@${code_size_new}@" "${TAR_OUT_DIR}/${MODULE_NAME}/${MODULE_NAME}_meta.json"

# get the final tar package.
chmod -R 700 "${TAR_OUT_DIR}"

cp -ar "${TAR_OUT_DIR}/"* "${OUTPUT_DIR}"
mkdir -p "$OUTPUT_DIR/templates/"
cp -arf "${PROJECT_DIR}/build/system-function-config.yaml" "${OUTPUT_DIR}/templates/system-function-config.yaml"
cp -arf "${PROJECT_DIR}/build/faasfrontend-function-config.yaml" "${OUTPUT_DIR}/templates/faasfrontend-function-config.yaml"
cp -arf "${PROJECT_DIR}/build/faasfrontend-function-meta.yaml" "${OUTPUT_DIR}/templates/faasfrontend-function-meta.yaml"
cp -arf "${PROJECT_DIR}/build/fassfrontend-service.yaml" "${OUTPUT_DIR}/templates/fassfrontend-service.yaml"
chmod -R 700 "${OUTPUT_DIR}"