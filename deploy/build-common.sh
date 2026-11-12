#!/bin/bash

# gis、file 的 build.sh 使用的镜像构建函数：准备基础镜像、构建、校验架构、导出 tar。

# 加载部署共享工具（架构解析与校验）
_BUILD_COMMON_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${_BUILD_COMMON_DIR}/common.sh"
source "${_BUILD_COMMON_DIR}/base-image/common.sh"

# 日志统一写 stderr：build_service_image 通过 stdout 返回镜像标签，build.sh 的
# BUILD_OUTPUT_ONLY 模式通过 stdout 返回 tar 文件名，stdout 只能承载返回值
log()   { echo -e "\033[1;34m[INFO]\033[0m $1" >&2; }
warn()  { echo -e "\033[1;33m[WARN]\033[0m $1" >&2; }
error() { echo -e "\033[1;31m[ERROR]\033[0m $1" >&2; }

# 构建业务镜像的核心函数
# 参数:
#   $1: 目标平台（如 linux/amd64）
#   $2: 镜像名称
#   $3: 镜像标签
#   $4: Dockerfile 路径
#   $5: 构建上下文路径
#   $6: 是否需要 GEOS_VERSION 参数（gis=true, file=false）
build_service_image() {
  local platform=${1:?platform required}
  local image_name=${2:?image_name required}
  local image_tag=${3:?image_tag required}
  local dockerfile=${4:?dockerfile required}
  local context=${5:?context required}
  local need_geos=${6:-false}

  local arch=$(target_arch "$platform")
  # 与 ensure.sh 保持一致：外部指定 BASE_IMAGE 时使用该镜像
  local base_img=${BASE_IMAGE:-$(base_image "$arch")}

  log "确认基础镜像: ${base_img} (${arch})"
  bash "${_BUILD_COMMON_DIR}/base-image/ensure.sh" "$arch" >&2 || {
    error "基础镜像准备失败: ${base_img}"
    return 1
  }

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

  log "构建镜像: ${full_tag} (linux/${arch})"
  DOCKER_BUILDKIT=1 docker build \
      --progress=plain \
      --platform "linux/${arch}" \
      "${build_args[@]}" \
      "${proxy_args[@]}" \
      -t "${full_tag}" \
      -f "${dockerfile}" "${context}" >&2 || {
    error "镜像构建失败: ${full_tag}"
    return 1
  }

  assert_image_arch "${full_tag}" "${arch}" || return 1
  log "镜像架构校验通过: ${full_tag} (${arch})"

  echo "${full_tag}"  # 返回完整标签供调用方使用
}

# 保存镜像为 tar 文件
# 参数:
#   $1: 镜像完整标签
#   $2: 输出 tar 文件名
save_image_tar() {
  local image_tag=${1:?image_tag required}
  local tar_name=${2:?tar_name required}

  log "保存镜像: ${tar_name}"
  docker save -o "${tar_name}" "${image_tag}" >&2 || {
    error "镜像保存失败: ${tar_name}"
    return 1
  }
}

# 清理本地镜像标签
# 参数:
#   $1: 镜像完整标签
cleanup_image_tag() {
  local image_tag=${1:?image_tag required}

  log "清理本地镜像标签: ${image_tag}"
  docker image rm "${image_tag}" >/dev/null 2>&1 || true
}
