#!/bin/bash

# 构建 gis/file 共用的基础镜像，默认同时构建 amd64 与 arm64。
# 只在 GEOS、ffmpeg 或 alpine 版本变化时需要重建。
#
# 用法:
#   bash deploy/base-image/build.sh
#   PLATFORMS="linux/amd64" bash deploy/base-image/build.sh

set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
# 构建上下文用 deploy/，便于直接引用 deploy/tls 下的共享证书
CONTEXT_DIR=$(cd "${SCRIPT_DIR}/.." && pwd)
source "${SCRIPT_DIR}/common.sh"

PLATFORMS=${PLATFORMS:-"linux/amd64 linux/arm64"}

log()   { echo -e "\033[1;34m[INFO]\033[0m $1"; }
error() { echo -e "\033[1;31m[ERROR]\033[0m $1"; }

PROXY_ARGS=()
if [ -n "$HTTP_PROXY" ]; then
  PROXY_ARGS+=(--build-arg HTTP_PROXY=$HTTP_PROXY)
fi
if [ -n "$HTTPS_PROXY" ]; then
  PROXY_ARGS+=(--build-arg HTTPS_PROXY=$HTTPS_PROXY)
fi

for PLATFORM in ${PLATFORMS}; do
  ARCH=$(target_arch "${PLATFORM}")
  IMAGE=$(base_image "${ARCH}")

  log "构建基础镜像: ${IMAGE} (${PLATFORM})"
  DOCKER_BUILDKIT=1 docker build \
      --platform "${PLATFORM}" \
      --progress=plain \
      --build-arg GEOS_VERSION=${BASE_GEOS_VERSION} \
      "${PROXY_ARGS[@]}" \
      -t "${IMAGE}" \
      -f "${SCRIPT_DIR}/Dockerfile" "${CONTEXT_DIR}" || {
    error "基础镜像构建失败: ${IMAGE}"
    exit 1
  }

  BUILT_ARCH=$(docker image inspect -f '{{.Architecture}}' "${IMAGE}" 2>/dev/null || echo "")
  if [ "${BUILT_ARCH}" != "${ARCH}" ]; then
    error "基础镜像架构异常: ${IMAGE} 实际为 '${BUILT_ARCH}'，期望 '${ARCH}'"
    exit 1
  fi
done

log "基础镜像构建完成 ✅"
for PLATFORM in ${PLATFORMS}; do
  log "  $(base_image "$(target_arch "${PLATFORM}")")"
done
