#!/bin/bash

# 确保当前目标架构的基础镜像存在，缺失时自动构建。
# 由 deploy/build-common.sh（gis、file 的 build.sh）在构建业务镜像前调用。
#
# 用法:
#   bash deploy/base-image/ensure.sh            # 默认架构（见 deploy/common.sh 的 DEFAULT_ARCH）
#   bash deploy/base-image/ensure.sh arm64      # 指定架构
#
# 若外部通过 BASE_IMAGE 指定了自定义镜像，本脚本只做存在性检查，不自动构建
# （build.sh 只会构建 zero-service/runtime:<版本>-<arch>）。

set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
source "${SCRIPT_DIR}/common.sh"

ARCH=${1:-$(target_arch)}
if [ -n "${BASE_IMAGE:-}" ]; then
  IMAGE=${BASE_IMAGE}
else
  IMAGE=$(base_image "${ARCH}")
fi

log()   { echo -e "\033[1;34m[INFO]\033[0m $1"; }
error() { echo -e "\033[1;31m[ERROR]\033[0m $1"; }

if docker image inspect "${IMAGE}" > /dev/null 2>&1; then
  log "基础镜像已存在: ${IMAGE}"
  exit 0
fi

if [ -n "${BASE_IMAGE:-}" ]; then
  error "自定义 BASE_IMAGE 不存在: ${IMAGE}"
  error "build.sh 只构建 zero-service/runtime:<版本>-<arch>，请先手动构建该镜像"
  exit 1
fi

log "基础镜像缺失，开始构建: ${IMAGE}"
PLATFORMS="linux/${ARCH}" bash "${SCRIPT_DIR}/build.sh"
