# runtime 基础镜像（base-image）

gis、file 运行期依赖 C 库与外部命令，目标服务器无外网，无法在部署时安装依赖。本目录提供统一的 runtime 基础镜像，业务镜像只在其上叠加二进制与配置。

## 镜像内容

| 项目 | 内容 |
| --- | --- |
| 镜像名 | `zero-service/runtime:<版本>-<arch>`，例如 `zero-service/runtime:1.0.0-amd64` |
| 基础系统 | Alpine 3.24 |
| C 运行库 | GEOS（默认 3.14.1），供 gis 使用 |
| 外部命令 | `ffmpeg`、`ffprobe`，供 file 使用 |
| 证书 | 系统 CA 证书 + [`deploy/tls/server.crt`](../tls/)（已加入系统信任库） |
| 时区 | `tzdata`，`TZ=Asia/Shanghai` |
| 架构 | `amd64`、`arm64`，以标签后缀区分 |

镜像名不体现内部依赖，新增 C 依赖时只修改 `Dockerfile` 并提升 `BASE_RUNTIME_VERSION`。

## 文件

| 文件 | 说明 |
| --- | --- |
| `Dockerfile` | 基础镜像定义，构建上下文为 `deploy/` |
| `common.sh` | `BASE_RUNTIME_VERSION`、`BASE_GEOS_VERSION` 与镜像命名函数 `base_image` |
| `build.sh` | 构建基础镜像，默认 `linux/amd64` 与 `linux/arm64`，构建后校验架构 |
| `ensure.sh` | 检查指定架构的基础镜像，缺失时调用 `build.sh` 构建 |

## 构建

```bash
bash deploy/base-image/build.sh                              # amd64 + arm64
PLATFORMS="linux/arm64" bash deploy/base-image/build.sh      # 单一架构

bash deploy/base-image/ensure.sh arm64                       # 缺失时构建
```

- gis、file 的 `build.sh` 经 `deploy/build-common.sh` 调用 `ensure.sh`，日常构建无需手动执行。
- 基础镜像通过 `apk` 安装依赖，构建非本机架构时需要 qemu/binfmt。
- `ensure.sh` 只按 `<版本>-<arch>` 判断镜像是否存在。修改基础镜像内容（GEOS 版本、Alpine 版本、证书等）时必须提升 `BASE_RUNTIME_VERSION`。
- 设置 `BASE_IMAGE` 时，`ensure.sh` 只检查该镜像是否存在，不自动构建。
- 基础镜像只保存在构建机本地，不推送仓库；目标服务器只接收业务镜像 tar 包。

## 证书

- `deploy/tls/server.crt` 复制到 `/usr/local/share/ca-certificates/zero-service.crt` 并执行 `update-ca-certificates`，容器内 HTTPS 客户端信任与 nginx 相同的自签证书。
- 私钥 `deploy/tls/server.key` 不进入构建上下文：[`deploy/.dockerignore`](../.dockerignore) 采用白名单，仅放行 `tls/server.crt`。需要私钥的服务在运行时挂载。
- 证书单独成层，更新证书只重建该层；证书属于镜像内容，更新后同样需要提升 `BASE_RUNTIME_VERSION`。

## 业务镜像使用方式

| 服务 | 构建阶段 | 跨架构构建 |
| --- | --- | --- |
| gis | `CGO_ENABLED=1`，安装 `gcc musl-dev pkgconf geos-dev`，`geos-dev` 版本取 `BASE_GEOS_VERSION` | 构建阶段按目标架构运行，需要 qemu |
| file | `CGO_ENABLED=0`，`--platform=$BUILDPLATFORM` 交叉编译 | 构建阶段不需要 qemu |

两者最终阶段均为 `FROM ${BASE_IMAGE}`，`BASE_IMAGE` 由 `build.sh` 按目标架构传入，Dockerfile 不设默认值。其余服务运行在 `scratch` 上，不使用本镜像。多平台构建规范见 [`deploy/MULTI_PLATFORM_BUILD.md`](../MULTI_PLATFORM_BUILD.md)。
