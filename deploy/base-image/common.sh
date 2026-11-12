#!/bin/bash

# 基础镜像命名与版本（本地构建，不带公司/仓库前缀）。
# 由 deploy/base-image/build.sh、ensure.sh 与 deploy/build-common.sh 共同使用；
# 架构解析（target_arch）与镜像校验（assert_image_arch）在 deploy/common.sh。

_COMMON_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${_COMMON_DIR}/../common.sh"

# 通用 runtime 基础环境版本：内容升级时 bump，不因新增某个 C 依赖而改名
BASE_RUNTIME_VERSION=${BASE_RUNTIME_VERSION:-1.0.0}
# 仅作为构建参数，控制基础环境内的 GEOS 版本，不进入镜像名
BASE_GEOS_VERSION=${BASE_GEOS_VERSION:-3.14.1}

# 业务运行基础镜像：通用 runtime 环境（当前含 GEOS、ffmpeg，后续可继续加 C 依赖）
base_image() {
  echo "zero-service/runtime:${BASE_RUNTIME_VERSION}-${1:?arch required}"
}
