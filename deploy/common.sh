#!/bin/bash

# 部署脚本共用工具：目标架构解析、镜像架构校验。
# 由各业务服务的 build.sh / deploy.sh、deploy/build-common.sh 与 deploy/base-image 脚本复用。

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
