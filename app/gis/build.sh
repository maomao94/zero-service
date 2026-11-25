#!/bin/bash

# gis 服务多平台镜像构建脚本（不含部署）。
# 构建逻辑统一在 deploy/service-build.sh，本脚本只声明运行基础镜像类型：CGO 链接 GEOS，运行在 env-runtime（base + GEOS/ffmpeg）。
# 用法: bash build.sh [环境] [平台...]，默认同时构建 linux/amd64 与 linux/arm64。
# 环境变量: BUILD_TIMESTAMP、BUILD_OUTPUT_ONLY、BASE_IMAGE，详见 deploy/MULTI_PLATFORM_BUILD.md。

set -e

SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
export SERVICE_DIR="${SCRIPT_DIR}" BASE_KIND=env NEED_GEOS=true
exec bash "${SCRIPT_DIR}/../../deploy/service-build.sh" "$@"
