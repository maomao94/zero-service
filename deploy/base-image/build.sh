#!/bin/bash

# 构建业务镜像的运行基础镜像，默认同时构建 amd64 与 arm64。
# 只在 Alpine 版本、依赖包或镜像内容变化时需要重建。
#
# 用法:
#   bash deploy/base-image/build.sh                              # runtime（gis/file），双架构
#   bash deploy/base-image/build.sh go                           # go-runtime（纯 Go 服务），双架构
#   PLATFORMS="linux/amd64" bash deploy/base-image/build.sh go   # 单一架构

set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
# 构建上下文用 deploy/，便于直接引用 deploy/tls 下的共享证书
CONTEXT_DIR=$(cd "${SCRIPT_DIR}/.." && pwd)
source "${SCRIPT_DIR}/common.sh"

KIND=${1:-runtime}
PLATFORMS=${PLATFORMS:-"linux/amd64 linux/arm64"}

log()   { echo -e "\033[1;34m[INFO]\033[0m $1"; }
error() { echo -e "\033[1;31m[ERROR]\033[0m $1"; }

case "${KIND}" in
  runtime)
    DOCKERFILE="${SCRIPT_DIR}/Dockerfile"
    BUILD_ARGS=(--build-arg "GEOS_VERSION=${BASE_GEOS_VERSION}")
    image_name() { base_image "$1"; }
    ;;
  go)
    DOCKERFILE="${SCRIPT_DIR}/go-runtime/Dockerfile"
    BUILD_ARGS=()
    image_name() { go_base_image "$1"; }
    ;;
  *)
    error "未知的基础镜像类型: ${KIND}（可选 runtime、go）"
    exit 1
    ;;
esac

PROXY_ARGS=()
if [ -n "$HTTP_PROXY" ]; then
  PROXY_ARGS+=(--build-arg HTTP_PROXY=$HTTP_PROXY)
fi
if [ -n "$HTTPS_PROXY" ]; then
  PROXY_ARGS+=(--build-arg HTTPS_PROXY=$HTTPS_PROXY)
fi

for PLATFORM in ${PLATFORMS}; do
  ARCH=$(target_arch "${PLATFORM}")
  IMAGE=$(image_name "${ARCH}")

  log "构建基础镜像: ${IMAGE} (${PLATFORM})"
  DOCKER_BUILDKIT=1 docker build \
      --platform "${PLATFORM}" \
      --progress=plain \
      "${BUILD_ARGS[@]}" \
      "${PROXY_ARGS[@]}" \
      -t "${IMAGE}" \
      -f "${DOCKERFILE}" "${CONTEXT_DIR}" || {
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
  log "  $(image_name "$(target_arch "${PLATFORM}")")"
done
