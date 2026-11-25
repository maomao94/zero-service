# 业务镜像基础环境（base-image）

业务镜像不在部署时安装依赖：目标服务器无外网，运行期依赖必须预置在镜像里。本目录维护两类基础镜像，业务 Dockerfile 只在其上叠加二进制与配置。

## 镜像

| 镜像 | 使用者 | 内容 |
| --- | --- | --- |
| `zero-service/runtime:<版本>-<arch>` | gis、file | Alpine 3.24、GEOS（默认 3.14.1）、`ffmpeg`/`ffprobe`、`bash`、系统 CA 证书 + [`deploy/tls/server.crt`](../tls/)（已加入系统信任库）、`tzdata` |
| `zero-service/go-runtime:<版本>-<arch>` | 其余纯 Go 业务服务（20 个） | Alpine 3.24、`bash`、`curl`、`procps`、`iproute2`、`tcpdump`、系统 CA 证书、`tzdata` |

- 架构以标签后缀区分：`amd64`、`arm64`。
- 两个镜像都提供 `/bin/bash`，现场排查可直接 `docker exec -it <容器> /bin/bash`。此前纯 Go 服务运行在 `scratch` 上、gis/file 未装 bash，同样的命令会报 `stat /bin/bash: no such file or directory`。
- 镜像名不体现内部依赖，新增依赖时只修改对应 Dockerfile 并提升版本变量。

## 文件

| 文件 | 说明 |
| --- | --- |
| `Dockerfile` | `zero-service/runtime` 定义，构建上下文为 `deploy/` |
| `go-runtime/Dockerfile` | `zero-service/go-runtime` 定义，构建上下文为 `deploy/` |
| `common.sh` | `BASE_RUNTIME_VERSION`、`BASE_GO_RUNTIME_VERSION`、`BASE_GEOS_VERSION` 与镜像命名函数 `base_image`、`go_base_image` |
| `build.sh` | 构建基础镜像，参数 `runtime`（默认）或 `go`；默认 `linux/amd64` 与 `linux/arm64`，构建后校验架构 |
| `ensure.sh` | 检查指定架构与类型的基础镜像，缺失时调用 `build.sh` 构建 |

## 构建

```bash
bash deploy/base-image/build.sh                              # runtime：amd64 + arm64
bash deploy/base-image/build.sh go                           # go-runtime：amd64 + arm64
PLATFORMS="linux/arm64" bash deploy/base-image/build.sh go   # 单一架构

bash deploy/base-image/ensure.sh arm64                       # runtime，缺失时构建
bash deploy/base-image/ensure.sh arm64 go                    # go-runtime，缺失时构建
```

- 业务服务的 `build.sh` 会调用 `ensure.sh`（gis/file 经 `deploy/build-common.sh` 传 `runtime`，纯 Go 服务传 `go`），日常构建无需手动执行。
- 基础镜像通过 `apk` 安装依赖，构建非本机架构时需要 qemu/binfmt。
- `ensure.sh` 只按 `<版本>-<arch>` 判断镜像是否存在。修改基础镜像内容（GEOS 版本、Alpine 版本、证书、排查命令）时必须提升对应版本变量：
  - `BASE_RUNTIME_VERSION`：`zero-service/runtime`
  - `BASE_GO_RUNTIME_VERSION`：`zero-service/go-runtime`
- 设置 `BASE_IMAGE` 时，`ensure.sh` 只检查该镜像是否存在，不自动构建；`BASE_IMAGE=scratch` 直接跳过，供纯 Go 服务构建无 shell 的精简镜像。
- 基础镜像只保存在构建机本地，不推送仓库；目标服务器只接收业务镜像 tar 包，`docker save` 会把基础镜像的层一并打入 tar，无需单独分发。

## 证书

- `deploy/tls/server.crt` 复制到 `/usr/local/share/ca-certificates/zero-service.crt` 并执行 `update-ca-certificates`（仅 `zero-service/runtime`），容器内 HTTPS 客户端信任与 nginx 相同的自签证书；`zero-service/go-runtime` 只带系统 CA 证书，行为与原先 `scratch` 镜像一致。
- 私钥 `deploy/tls/server.key` 不进入构建上下文：[`deploy/.dockerignore`](../.dockerignore) 采用白名单，仅放行 `tls/server.crt`。需要私钥的服务在运行时挂载。
- 证书单独成层，更新证书只重建该层；证书属于镜像内容，更新后同样需要提升 `BASE_RUNTIME_VERSION`。

## 业务镜像使用方式

| 服务 | 构建阶段 | 跨架构构建 |
| --- | --- | --- |
| gis | `CGO_ENABLED=1`，安装 `gcc musl-dev pkgconf geos-dev`，`geos-dev` 版本取 `BASE_GEOS_VERSION` | 构建阶段按目标架构运行，需要 qemu |
| file | `CGO_ENABLED=0`，`--platform=$BUILDPLATFORM` 交叉编译 | 构建阶段不需要 qemu |
| 其余纯 Go 服务 | `CGO_ENABLED=0`，`--platform=$BUILDPLATFORM` 交叉编译 | 构建阶段不需要 qemu |

最终阶段统一为 `FROM ${BASE_IMAGE}`，`BASE_IMAGE` 由 `build.sh` 按目标架构传入，Dockerfile 不设默认值。多平台构建规范见 [`deploy/MULTI_PLATFORM_BUILD.md`](../MULTI_PLATFORM_BUILD.md)。

## 体积与精简

`zero-service/go-runtime` 让单个业务镜像的体积增加约 33 MB（`docker image ls` 显示的内容大小），换来的能力是容器内可进入、可抓包、可验证上游接口。需要最小体积时：

- 单个服务临时精简：`BASE_IMAGE=scratch bash build.sh <环境>`。此时镜像回到只有二进制与 `etc/`，无法 `docker exec`。
- 长期精简：裁剪 `go-runtime/Dockerfile` 的 `apk add` 列表并提升 `BASE_GO_RUNTIME_VERSION`，不要只改本地镜像内容。
