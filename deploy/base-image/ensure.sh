#!/bin/bash

# 确保当前目标架构的基础镜像存在，缺失时自动构建。
# 由各业务服务的 build.sh 在构建镜像前调用。
#
# 用法:
#   bash deploy/base-image/ensure.sh                # runtime，默认架构（见 deploy/common.sh 的 DEFAULT_ARCH）
#   bash deploy/base-image/ensure.sh arm64          # runtime，指定架构
#   bash deploy/base-image/ensure.sh arm64 go       # go-runtime（纯 Go 业务服务）
#
# 若外部通过 BASE_IMAGE 指定了自定义镜像，本脚本只做存在性检查，不自动构建
# （build.sh 只会构建 zero-service/runtime 与 zero-service/go-runtime）。
# BASE_IMAGE=scratch 时直接跳过，用于构建无 shell 的精简镜像。

set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
source "${SCRIPT_DIR}/common.sh"

ARCH=${1:-$(target_arch)}
KIND=${2:-runtime}

log()   { echo -e "\033[1;34m[INFO]\033[0m $1"; }
error() { echo -e "\033[1;31m[ERROR]\033[0m $1"; }

case "${KIND}" in
  runtime) DEFAULT_IMAGE=$(base_image "${ARCH}") ;;
  go)      DEFAULT_IMAGE=$(go_base_image "${ARCH}") ;;
  *)
    error "未知的基础镜像类型: ${KIND}（可选 runtime、go）"
    exit 1
    ;;
esac

IMAGE=${BASE_IMAGE:-${DEFAULT_IMAGE}}

if [ "${IMAGE}" = "scratch" ]; then
  log "使用 scratch，跳过基础镜像准备"
  exit 0
fi

if docker image inspect "${IMAGE}" > /dev/null 2>&1; then
  log "基础镜像已存在: ${IMAGE}"
  exit 0
fi

if [ -n "${BASE_IMAGE:-}" ]; then
  error "自定义 BASE_IMAGE 不存在: ${IMAGE}"
  error "build.sh 只构建 zero-service/runtime 与 zero-service/go-runtime，请先手动构建该镜像"
  exit 1
fi

log "基础镜像缺失，开始构建: ${IMAGE}"
PLATFORMS="linux/${ARCH}" bash "${SCRIPT_DIR}/build.sh" "${KIND}"
