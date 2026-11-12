#!/bin/bash

# socketpush 服务多平台镜像构建脚本（不含部署）
# 专门用于生产环境多平台镜像交付场景
#
# 用法:
#   bash build.sh [环境] [平台...]
#   bash build.sh test                           # 默认同时构建 amd64 与 arm64
#   bash build.sh test linux/amd64               # 只构建 amd64
#   bash build.sh test linux/arm64               # 只构建 arm64
#   bash build.sh test "linux/amd64 linux/arm64" # 显式指定双架构
#
# 输出：当前目录下生成 <镜像名>_<arch>_<timestamp>.tar 文件，可直接 docker load
#
# 环境变量：
#   BUILD_TIMESTAMP - 指定时间戳（用于 deploy.sh 调用时保持一致性）
#   BUILD_OUTPUT_ONLY - 设为 true 时只输出 tar 文件名到 stdout，日志输出到 stderr

set -e

ENVIRONMENT=${1:-dev}
PLATFORMS=${2:-"linux/amd64 linux/arm64"}
ENV_FILE="env/${ENVIRONMENT}.env"

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
source "${SCRIPT_DIR}/../../deploy/common.sh"

# BUILD_OUTPUT_ONLY=true 时，日志输出到 stderr
if [ "${BUILD_OUTPUT_ONLY}" = "true" ]; then
  log()   { echo -e "\033[1;34m[INFO]\033[0m $1" >&2; }
  warn()  { echo -e "\033[1;33m[WARN]\033[0m $1" >&2; }
  error() { echo -e "\033[1;31m[ERROR]\033[0m $1" >&2; }
else
  log()   { echo -e "\033[1;34m[INFO]\033[0m $1"; }
  warn()  { echo -e "\033[1;33m[WARN]\033[0m $1"; }
  error() { echo -e "\033[1;31m[ERROR]\033[0m $1"; }
fi

# 检查 .env 文件
if [ ! -f "$ENV_FILE" ]; then
  error "环境文件不存在: $ENV_FILE"
  exit 1
fi

log "加载环境变量: $ENV_FILE"
export $(grep -v '^#' "$ENV_FILE" | xargs)

# 只需要 IMAGE_NAME，不需要部署相关变量
REQUIRED_VARS=(IMAGE_NAME)
for var in "${REQUIRED_VARS[@]}"; do
  if [ -z "${!var}" ]; then
    error "缺少必要环境变量: $var"
    exit 1
  fi
done

# 支持外部指定时间戳（deploy.sh 调用时使用）
TIMESTAMP=${BUILD_TIMESTAMP:-$(date +%Y%m%d%H%M%S)}
SANITIZED_IMAGE_NAME=$(echo "$IMAGE_NAME" | tr '/' '-')

# 准备代理参数
PROXY_ARGS=()
[ -n "$HTTP_PROXY" ]  && PROXY_ARGS+=(--build-arg "HTTP_PROXY=$HTTP_PROXY")
[ -n "$HTTPS_PROXY" ] && PROXY_ARGS+=(--build-arg "HTTPS_PROXY=$HTTPS_PROXY")

BUILT_TARS=()

for PLATFORM in ${PLATFORMS}; do
  ARCH=$(target_arch "$PLATFORM")
  TAG="${TIMESTAMP}-${ARCH}"
  TAR_NAME="${SANITIZED_IMAGE_NAME}_${ARCH}_${TIMESTAMP}.tar"

  log "=========================================="
  log "开始构建 ${ARCH} 架构镜像"
  log "=========================================="

  log "构建镜像: ${IMAGE_NAME}:${TAG} (linux/${ARCH})"
  DOCKER_BUILDKIT=1 docker build \
      --progress=plain \
      --platform "linux/${ARCH}" \
      "${PROXY_ARGS[@]}" \
      -t "${IMAGE_NAME}:${TAG}" \
      -f ./Dockerfile ../../ >&2 || {
    error "镜像构建失败: ${ARCH}"
    exit 1
  }

  assert_image_arch "${IMAGE_NAME}:${TAG}" "${ARCH}"
  log "镜像架构校验通过: ${IMAGE_NAME}:${TAG} (${ARCH})"

  log "保存镜像: ${TAR_NAME}"
  docker save -o "${TAR_NAME}" "${IMAGE_NAME}:${TAG}" >&2

  log "清理本地镜像标签: ${IMAGE_NAME}:${TAG}"
  docker image rm "${IMAGE_NAME}:${TAG}" >/dev/null

  BUILT_TARS+=("${TAR_NAME}")
done

log "=========================================="
log "所有镜像构建完成 ✅"
log "=========================================="

# BUILD_OUTPUT_ONLY=true 时，输出 tar 文件名到 stdout（一行一个）
if [ "${BUILD_OUTPUT_ONLY}" = "true" ]; then
  for tar in "${BUILT_TARS[@]}"; do
    echo "$tar"
  done
else
  for tar in "${BUILT_TARS[@]}"; do
    TAR_SIZE=$(du -h "$tar" | cut -f1)
    log "  ${tar} (${TAR_SIZE})"
  done
fi
