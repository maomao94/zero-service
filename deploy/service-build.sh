#!/bin/bash

# 业务服务镜像构建的统一实现（不含部署），由各服务目录的 build.sh 调用。
# 各服务的差异只通过环境变量声明，服务目录只保留一个薄壳脚本：
#   SERVICE_DIR - 服务目录，由服务 build.sh 的 SCRIPT_DIR 传入
#   BASE_KIND   - 运行基础镜像类型：base（纯 Go，默认）| env（GEOS/ffmpeg，file/gis）
#   NEED_GEOS   - 是否向 Dockerfile 传 GEOS_VERSION（gis=true），默认 false
#
# 用法（经服务目录 build.sh 透传）:
#   bash build.sh [环境] [平台...]
#   bash build.sh test                           # 默认同时构建 amd64 与 arm64
#   bash build.sh test linux/amd64               # 只构建 amd64
#   bash build.sh test linux/arm64               # 只构建 arm64
#   bash build.sh test "linux/amd64 linux/arm64" # 显式指定双架构
#
# 输出：服务目录下生成 <镜像名>_<arch>_<timestamp>.tar，可直接 docker load
#
# 环境变量：
#   BUILD_TIMESTAMP  - 指定时间戳（用于 deploy.sh 调用时保持一致性）
#   BUILD_OUTPUT_ONLY - 设为 true 时只输出 tar 文件名到 stdout，日志输出到 stderr
#   BASE_IMAGE       - 覆盖运行基础镜像；纯 Go 默认 zero-service/base-runtime:<版本>-<arch>，
#                      file/gis 默认 zero-service/env-runtime:<版本>-<arch>；
#                      设为 scratch 可构建无 shell 的精简镜像

set -e

SERVICE_DIR=${SERVICE_DIR:?SERVICE_DIR required}
BASE_KIND=${BASE_KIND:-base}
NEED_GEOS=${NEED_GEOS:-false}

ENVIRONMENT=${1:-dev}
PLATFORMS=${2:-"linux/amd64 linux/arm64"}
ENV_FILE="${SERVICE_DIR}/env/${ENVIRONMENT}.env"

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
# 构建上下文固定为仓库根目录，与服务所在位置无关
CONTEXT_DIR=$(cd "${SERVICE_DIR}/../.." && pwd)
source "${SCRIPT_DIR}/common.sh"

# BUILD_OUTPUT_ONLY=true 时，日志输出到 stderr，stdout 只承载 tar 文件名
if [ "${BUILD_OUTPUT_ONLY}" = "true" ]; then
  log()   { echo -e "\033[1;34m[INFO]\033[0m $1" >&2; }
  warn()  { echo -e "\033[1;33m[WARN]\033[0m $1" >&2; }
  error() { echo -e "\033[1;31m[ERROR]\033[0m $1" >&2; }
else
  log()   { echo -e "\033[1;34m[INFO]\033[0m $1"; }
  warn()  { echo -e "\033[1;33m[WARN]\033[0m $1"; }
  error() { echo -e "\033[1;31m[ERROR]\033[0m $1"; }
fi

case "${BASE_KIND}" in
  base|env) ;;
  *)
    error "无效的 BASE_KIND: ${BASE_KIND}（可选 base、env）"
    exit 1
    ;;
esac

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

BUILT_TARS=()

for PLATFORM in ${PLATFORMS}; do
  ARCH=$(target_arch "$PLATFORM")
  TAG="${TIMESTAMP}-${ARCH}"
  TAR_NAME="${SANITIZED_IMAGE_NAME}_${ARCH}_${TIMESTAMP}.tar"

  log "=========================================="
  log "开始构建 ${ARCH} 架构镜像"
  log "=========================================="

  if ! FULL_TAG=$(build_service_image \
      "$PLATFORM" \
      "$IMAGE_NAME" \
      "$TAG" \
      "${SERVICE_DIR}/Dockerfile" \
      "$CONTEXT_DIR" \
      "$NEED_GEOS" \
      "$BASE_KIND"); then
    error "镜像构建失败: ${ARCH}"
    exit 1
  fi

  save_image_tar "$FULL_TAG" "${SERVICE_DIR}/${TAR_NAME}" || {
    error "镜像保存失败: ${ARCH}"
    exit 1
  }
  cleanup_image_tag "$FULL_TAG"

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
    TAR_SIZE=$(du -h "${SERVICE_DIR}/${tar}" | cut -f1)
    log "  ${tar} (${TAR_SIZE})"
  done
fi
