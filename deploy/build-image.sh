#!/bin/bash

# 构建业务镜像的运行基础镜像（定义见 base-image/Dockerfile）。
#   base：zero-service/base-runtime，bash + 排查命令 + CA 证书（含内部自签）+ 时区，
#         所有业务镜像的最小底座，保证容器内能 docker exec -it <容器> /bin/bash
#   env ：zero-service/env-runtime，base + GEOS + ffmpeg/ffprobe，供 gis、file 使用
# 只在 Alpine 版本、依赖包、证书或版本变量变化时需要重建；业务 build.sh 会在缺失时自动调用本脚本。
#
# 用法:
#   bash deploy/build-image.sh                       # base + env，amd64 + arm64
#   bash deploy/build-image.sh base                  # 只构建 base-runtime
#   bash deploy/build-image.sh env                   # 只构建 env-runtime（自动先建 base）
#   bash deploy/build-image.sh base linux/arm64      # 指定单个平台
#   PLATFORMS="linux/arm64" bash deploy/build-image.sh env

set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
# 构建上下文用 deploy/，便于直接引用 deploy/tls 下的共享证书；.dockerignore 采用白名单
CONTEXT_DIR=${SCRIPT_DIR}
source "${SCRIPT_DIR}/common.sh"

KIND=${1:-all}
PLATFORMS=${2:-${PLATFORMS:-"linux/amd64 linux/arm64"}}

log()   { echo -e "\033[1;34m[INFO]\033[0m $1"; }
error() { echo -e "\033[1;31m[ERROR]\033[0m $1"; }

# env 基于 base，只建 env 时先确保 base 存在
case "${KIND}" in
  base) KINDS=(base) ;;
  env)  KINDS=(base env) ;;
  all)  KINDS=(base env) ;;
  *)
    error "未知的基础镜像类型: ${KIND}（可选 base、env、all）"
    exit 1
    ;;
esac

PROXY_ARGS=()
if [ -n "$HTTP_PROXY" ]; then
  PROXY_ARGS+=(--build-arg "HTTP_PROXY=$HTTP_PROXY")
fi
if [ -n "$HTTPS_PROXY" ]; then
  PROXY_ARGS+=(--build-arg "HTTPS_PROXY=$HTTPS_PROXY")
fi

for BUILD_KIND in "${KINDS[@]}"; do
  for PLATFORM in ${PLATFORMS}; do
    ARCH=$(target_arch "${PLATFORM}")
    IMAGE=$(_runtime_image "${BUILD_KIND}" "${ARCH}")

    BUILD_ARGS=(--target "${BUILD_KIND}")
    if [ "${BUILD_KIND}" = "env" ]; then
      BUILD_ARGS+=(--build-arg "GEOS_VERSION=${BASE_GEOS_VERSION}")
    fi

    log "构建基础镜像: ${IMAGE} (${PLATFORM})"
    DOCKER_BUILDKIT=1 docker build \
        --platform "${PLATFORM}" \
        --progress=plain \
        "${BUILD_ARGS[@]}" \
        "${PROXY_ARGS[@]}" \
        -t "${IMAGE}" \
        -f "${SCRIPT_DIR}/base-image/Dockerfile" "${CONTEXT_DIR}" || {
      error "基础镜像构建失败: ${IMAGE}"
      exit 1
    }

    assert_image_arch "${IMAGE}" "${ARCH}"

    # 原生架构下冒烟验证：bash 可执行、CA 证书可用，确保现场能 exec 进容器
    if [ "$(normalize_arch "$(uname -m)")" = "${ARCH}" ]; then
      docker run --rm "${IMAGE}" /bin/bash -c \
          'test -x /bin/bash && test -s /etc/ssl/certs/ca-certificates.crt' || {
        error "基础镜像冒烟验证失败: ${IMAGE}"
        exit 1
      }
      log "冒烟验证通过: ${IMAGE} (bash + CA 证书)"
    fi
  done
done

log "基础镜像构建完成 ✅"
for BUILD_KIND in "${KINDS[@]}"; do
  for PLATFORM in ${PLATFORMS}; do
    log "  $(_runtime_image "${BUILD_KIND}" "$(target_arch "${PLATFORM}")")"
  done
done
