# 业务镜像基础环境（base-image）

业务镜像不在部署时安装依赖：目标服务器无外网，运行期依赖必须预置在镜像里。本目录只有一个 `Dockerfile`，通过两个 `target` 产出两层基础镜像，业务 Dockerfile 只在其上叠加二进制与配置。

## 镜像

| 镜像 | 使用者 | 内容 | 体积 |
| --- | --- | --- | --- |
| `zero-service/base-runtime:<版本>-<arch>` | 其余 20 个纯 Go 业务服务 | Alpine 3.24、`bash`、`curl`、`procps`、`iproute2`、`tcpdump`、系统 CA 证书 + [`deploy/tls/server.crt`](../tls/)（已加入系统信任库）、`tzdata` | 约 33 MB |
| `zero-service/env-runtime:<版本>-<arch>` | gis、file | `base-runtime` 全部内容 + GEOS（默认 3.14.1）、`ffmpeg`/`ffprobe` | 约 225 MB |

- **`base-runtime` 是所有业务镜像的硬底座**：必须能 `docker exec -it <容器> /bin/bash`，并具备 `curl`、`ss`、`ip`、`tcpdump` 等现场排查能力。`build-image.sh` 在原生架构构建后会跑一次冒烟验证，确认 bash 与 CA 证书可用。
- `env-runtime` 在 `base-runtime` 之上单独一层，只给 gis、file 使用，纯 Go 服务不会承担 GEOS/ffmpeg 的体积。
- 架构以标签后缀区分：`amd64`、`arm64`。
- 镜像名不体现内部依赖，新增依赖时修改对应 `target` 并提升版本变量。

## 文件

| 文件 | 说明 |
| --- | --- |
| `Dockerfile` | 单文件双 target：`base` → `zero-service/base-runtime`，`env` → `zero-service/env-runtime`；构建上下文为 `deploy/` |
| `README.md` | 本文件 |

构建入口和公共库不放在本目录：

| 文件 | 说明 |
| --- | --- |
| [`../build-image.sh`](../build-image.sh) | 基础镜像构建入口，参数 `base`、`env` 或 `all`（默认） |
| [`../service-build.sh`](../service-build.sh) | 业务镜像构建的唯一实现，各服务 `build.sh` 只是声明 `BASE_KIND`/`NEED_GEOS` 的薄壳 |
| [`../common.sh`](../common.sh) | 唯一公共库：镜像版本变量、镜像命名函数、`ensure_base_image`、业务镜像构建函数 |

## 构建

```bash
bash deploy/build-image.sh                       # base + env，amd64 + arm64
bash deploy/build-image.sh base                  # 只建 base-runtime
bash deploy/build-image.sh env                   # 只建 env-runtime（自动先建 base）
bash deploy/build-image.sh base linux/arm64      # 指定单个平台
PLATFORMS="linux/arm64" bash deploy/build-image.sh env
```

- 业务服务的 `build.sh` 经 `deploy/service-build.sh` 调用 `ensure_base_image` 检查基础镜像，缺失时自动调用 `build-image.sh`，日常构建无需手动执行。
- 基础镜像通过 `apk` 安装依赖，构建非本机架构时需要 qemu/binfmt。
- 基础镜像只保存在构建机本地，不推送仓库；目标服务器只接收业务镜像 tar 包，`docker save` 会把基础镜像的层一并打入 tar，无需单独分发。
- 设置 `BASE_IMAGE` 时，`ensure_base_image` 只检查该镜像是否存在，不自动构建；`BASE_IMAGE=scratch` 直接跳过，供纯 Go 服务构建无 shell 的精简镜像。

## 版本

版本变量在 [`../common.sh`](../common.sh) 中维护：

| 变量 | 对应镜像 |
| --- | --- |
| `BASE_RUNTIME_VERSION` | `zero-service/base-runtime` |
| `BASE_ENV_VERSION` | `zero-service/env-runtime` |
| `BASE_GEOS_VERSION` | `env` 阶段与 gis 构建阶段的 GEOS 版本，不进入镜像名 |

`ensure_base_image` 只按 `<版本>-<arch>` 判断镜像是否存在。修改基础镜像内容（依赖包、Alpine 版本、证书、排查命令）时必须提升对应版本变量：

- 只改 `env` 阶段：提升 `BASE_ENV_VERSION`。
- 改 `base` 阶段：`BASE_RUNTIME_VERSION` 与 `BASE_ENV_VERSION` 都要提升，因为 `env-runtime` 基于它构建。

## 证书

- `deploy/tls/server.crt` 复制到 `/usr/local/share/ca-certificates/zero-service.crt` 并执行 `update-ca-certificates`，两层镜像都信任与 nginx 相同的自签证书，调用内部 HTTPS 服务无需额外挂载。
- 私钥 `deploy/tls/server.key` 不进入构建上下文：[`deploy/.dockerignore`](../.dockerignore) 采用白名单，仅放行 `tls/server.crt`。需要私钥的服务在运行时挂载。
- 证书单独成层，更新证书只重建该层；证书属于镜像内容，更新后同样需要提升版本变量（改 `base` 阶段时两个版本都提升）。

## 业务镜像使用方式

| 服务 | 构建阶段 | 运行镜像 | 跨架构构建 |
| --- | --- | --- | --- |
| gis | `CGO_ENABLED=1`，安装 `gcc musl-dev pkgconf geos-dev`，`geos-dev` 版本取 `BASE_GEOS_VERSION` | `env-runtime` | 构建阶段按目标架构运行，需要 qemu |
| file | `CGO_ENABLED=0`，`--platform=$BUILDPLATFORM` 交叉编译 | `env-runtime` | 构建阶段不需要 qemu |
| 其余纯 Go 服务 | `CGO_ENABLED=0`，`--platform=$BUILDPLATFORM` 交叉编译 | `base-runtime` | 构建阶段不需要 qemu |

最终阶段统一为 `FROM ${BASE_IMAGE}`，`BASE_IMAGE` 由 `build.sh` 经 `ensure_base_image` 按目标架构传入，Dockerfile 不设默认值。多平台构建规范见 [`deploy/MULTI_PLATFORM_BUILD.md`](../MULTI_PLATFORM_BUILD.md)。

## 体积与精简

- `base-runtime` 让单个纯 Go 业务镜像增加约 33 MB（`docker image ls` 显示的内容大小），换来容器内可进入、可抓包、可验证上游接口。
- 需要最小体积时，单个服务临时用 `BASE_IMAGE=scratch bash build.sh <环境>`，镜像回到只有二进制与 `etc/`，无法 `docker exec`。
- 长期精简请裁剪 `Dockerfile` 中 `base` 阶段的 `apk add` 列表并提升 `BASE_RUNTIME_VERSION`，不要只改本地镜像内容。
