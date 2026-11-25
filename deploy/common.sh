#!/bin/bash

# 部署脚本唯一公共库：架构解析、运行基础镜像定义与准备、业务镜像构建、镜像归档。
# 由 deploy/service-build.sh、deploy/build-image.sh 与各业务服务的 deploy.sh 共同 source。
# 本文件只定义变量和函数，不定义 log/warn/error，避免覆盖调用方自己的日志函数。

_COMMON_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# 默认目标架构：现场服务器统一为 amd64，ARM 环境需显式指定 PLATFORM
DEFAULT_ARCH=${DEFAULT_ARCH:-amd64}

# 将 uname -m 或 Docker 的架构名归一为 Docker 平台架构：x86_64 -> amd64，aarch64 -> arm64
normalize_arch() {
  case "${1:-}" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) echo "${1:-}" ;;
  esac
}

# 解析目标架构：linux/arm64 -> arm64，linux/arm64/v8 -> arm64；单独传 arm64 也适用。
# 未传时默认 amd64，避免在 ARM 开发机上误构建
target_arch() {
  local platform=${1:-}
  if [ -z "$platform" ]; then
    echo "${DEFAULT_ARCH}"
    return
  fi
  local arch=${platform#*/}   # 去掉 os/ 前缀；不带 / 时保持原值
  echo "${arch%%/*}"          # 去掉 /v8 之类的 variant
}

# 校验镜像架构与目标一致，避免把错架构的镜像传到服务器（启动时才会 exec format error）
assert_image_arch() {
  local image=${1:?image required}
  local expected=${2:?arch required}
  local actual
  actual=$(docker image inspect -f '{{.Architecture}}' "${image}" 2>/dev/null || echo "")
  if [ "${actual}" != "${expected}" ]; then
    echo -e "\033[1;31m[ERROR]\033[0m 镜像架构 '${actual:-unknown}' 与目标架构 '${expected}' 不一致: ${image}" >&2
    exit 1
  fi
}

# 校验远端 Docker 架构与构建目标一致，避免把错架构镜像推到服务器
# 参数: host port user passwd expected_arch
# 镜像架构不匹配时容器启动才报 exec format error，因此在构建前先确认目标服务器架构。
assert_remote_arch() {
  local host=${1:?host required}
  local port=${2:?port required}
  local user=${3:?user required}
  local pass=${4:?pass required}
  local expected=${5:?arch required}
  local actual

  # 优先读取 Docker daemon 架构；daemon 不可用时回退 uname -m，再统一归一
  if ! actual=$(sshpass -p "${pass}" ssh -p "${port}" \
        -o StrictHostKeyChecking=no -o ConnectTimeout=10 \
        "${user}@${host}" \
        "docker version --format '{{.Server.Arch}}' 2>/dev/null || uname -m" 2>/dev/null); then
    echo -e "\033[1;31m[ERROR]\033[0m 无法获取远端架构: ${user}@${host}:${port}" >&2
    exit 1
  fi

  actual=$(normalize_arch "$(printf '%s\n' "${actual}" | tail -n 1 | tr -d '[:space:]')")
  if [ "${actual}" != "${expected}" ]; then
    echo -e "\033[1;31m[ERROR]\033[0m 目标架构 '${expected}' 与远端 '${user}@${host}' 架构 '${actual:-unknown}' 不一致" >&2
    echo -e "\033[1;31m[ERROR]\033[0m 请按服务器架构指定平台参数，例如: bash deploy.sh <环境> linux/${actual:-arm64}" >&2
    exit 1
  fi
}

# ==================== 运行基础镜像 ====================
# 两层结构（定义见 deploy/base-image/Dockerfile，构建入口 deploy/build-image.sh）：
#   base-runtime：bash + 排查命令 + CA 证书（含 deploy/tls 内部自签）+ 时区，
#                 所有业务镜像的最小底座，必须保证能 docker exec -it <容器> /bin/bash
#   env-runtime： base-runtime + GEOS + ffmpeg/ffprobe，供 gis、file 使用
# 版本只在镜像内容变化时提升，镜像名不体现内部依赖。
BASE_RUNTIME_VERSION=${BASE_RUNTIME_VERSION:-1.0.0}
BASE_ENV_VERSION=${BASE_ENV_VERSION:-1.0.0}
# 仅作为构建参数，控制基础镜像内的 GEOS 版本，不进入镜像名
BASE_GEOS_VERSION=${BASE_GEOS_VERSION:-3.14.1}

# base-runtime 镜像名：$1 架构
base_runtime_image() {
  echo "zero-service/base-runtime:${BASE_RUNTIME_VERSION}-${1:?arch required}"
}

# env-runtime 镜像名：$1 架构
env_runtime_image() {
  echo "zero-service/env-runtime:${BASE_ENV_VERSION}-${1:?arch required}"
}

# 按类型解析默认镜像名，未知类型写到 stderr 并返回非 0
# 参数: $1 base|env，$2 架构
_runtime_image() {
  case "${1:?kind required}" in
    base) base_runtime_image "${2:?arch required}" ;;
    env)  env_runtime_image "${2:?arch required}" ;;
    *)
      echo -e "\033[1;31m[ERROR]\033[0m 未知的基础镜像类型: ${1}（可选 base、env）" >&2
      return 1
      ;;
  esac
}

# 确保运行基础镜像存在，缺失时自动构建。
# 成功时把最终使用的镜像名写到 stdout，供调用方直接传给 docker build 的 BASE_IMAGE。
# 日志统一写 stderr，不污染调用方的 stdout（BUILD_OUTPUT_ONLY 模式只允许 stdout 返回 tar 名）。
# 参数: $1 base|env，$2 目标架构（默认 DEFAULT_ARCH）
# 环境变量 BASE_IMAGE：覆盖默认镜像，只检查存在性、不自动构建；设为 scratch 时跳过检查。
ensure_base_image() {
  local kind=${1:?kind required}
  local arch=${2:-$(target_arch)}
  local default_image image

  default_image=$(_runtime_image "${kind}" "${arch}") || return 1
  image=${BASE_IMAGE:-${default_image}}

  if [ "${image}" = "scratch" ]; then
    echo -e "\033[1;34m[INFO]\033[0m 使用 scratch，跳过基础镜像准备" >&2
    echo "${image}"
    return 0
  fi

  if docker image inspect "${image}" >/dev/null 2>&1; then
    echo -e "\033[1;34m[INFO]\033[0m 基础镜像已存在: ${image}" >&2
    echo "${image}"
    return 0
  fi

  if [ -n "${BASE_IMAGE:-}" ]; then
    echo -e "\033[1;31m[ERROR]\033[0m 自定义 BASE_IMAGE 不存在: ${image}" >&2
    echo -e "\033[1;31m[ERROR]\033[0m build-image.sh 只构建 zero-service/base-runtime 与 zero-service/env-runtime，请先手动构建该镜像" >&2
    return 1
  fi

  echo -e "\033[1;34m[INFO]\033[0m 基础镜像缺失，开始构建: ${image}" >&2
  PLATFORMS="linux/${arch}" bash "${_COMMON_DIR}/build-image.sh" "${kind}" >&2 || return 1
  echo "${image}"
}

# 业务镜像构建函数：准备基础镜像、docker build、架构校验。
# 日志写 stderr，stdout 只返回镜像完整标签，供 build.sh 在 BUILD_OUTPUT_ONLY 模式下保持返回契约。
# 参数:
#   $1 目标平台（如 linux/amd64）
#   $2 镜像名称
#   $3 镜像标签
#   $4 Dockerfile 路径
#   $5 构建上下文路径
#   $6 是否需要 GEOS_VERSION 参数（gis=true, file=false）
#   $7 运行基础镜像类型 base|env，默认 env
build_service_image() {
  local platform=${1:?platform required}
  local image_name=${2:?image_name required}
  local image_tag=${3:?image_tag required}
  local dockerfile=${4:?dockerfile required}
  local context=${5:?context required}
  local need_geos=${6:-false}
  local base_kind=${7:-env}
  local arch base_img

  arch=$(target_arch "${platform}")

  _build_log "准备运行基础镜像: ${base_kind} (${arch})"
  if ! base_img=$(ensure_base_image "${base_kind}" "${arch}"); then
    _build_error "基础镜像准备失败"
    return 1
  fi

  local full_tag="${image_name}:${image_tag}"

  # 准备代理参数
  local proxy_args=()
  [ -n "$HTTP_PROXY" ]  && proxy_args+=(--build-arg "HTTP_PROXY=$HTTP_PROXY")
  [ -n "$HTTPS_PROXY" ] && proxy_args+=(--build-arg "HTTPS_PROXY=$HTTPS_PROXY")

  # 准备构建参数
  local build_args=(
    --build-arg "BASE_IMAGE=${base_img}"
  )

  # gis 服务需要 GEOS_VERSION
  if [ "$need_geos" = "true" ]; then
    build_args+=(--build-arg "GEOS_VERSION=${BASE_GEOS_VERSION}")
  fi

  _build_log "构建镜像: ${full_tag} (linux/${arch})"
  DOCKER_BUILDKIT=1 docker build \
      --progress=plain \
      --platform "linux/${arch}" \
      "${build_args[@]}" \
      "${proxy_args[@]}" \
      -t "${full_tag}" \
      -f "${dockerfile}" "${context}" >&2 || {
    _build_error "镜像构建失败: ${full_tag}"
    return 1
  }

  assert_image_arch "${full_tag}" "${arch}" || return 1
  _build_log "镜像架构校验通过: ${full_tag} (${arch})"

  echo "${full_tag}"  # 返回完整标签供调用方使用
}

# 保存镜像为 tar 文件
# 参数: $1 镜像完整标签，$2 输出 tar 文件名
save_image_tar() {
  local image_tag=${1:?image_tag required}
  local tar_name=${2:?tar_name required}

  _build_log "保存镜像: ${tar_name}"
  docker save -o "${tar_name}" "${image_tag}" >&2 || {
    _build_error "镜像保存失败: ${tar_name}"
    return 1
  }
}

# 清理本地镜像标签
# 参数: $1 镜像完整标签
cleanup_image_tag() {
  local image_tag=${1:?image_tag required}

  _build_log "清理本地镜像标签: ${image_tag}"
  docker image rm "${image_tag}" >/dev/null 2>&1 || true
}

# 内部日志：所有构建函数共用，统一写 stderr，避免覆盖调用方的 log 函数
_build_log()   { echo -e "\033[1;34m[INFO]\033[0m $1" >&2; }
_build_error() { echo -e "\033[1;31m[ERROR]\033[0m $1" >&2; }
