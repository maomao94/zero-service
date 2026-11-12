# 多平台镜像构建与部署

本文档定义业务服务在 `linux/amd64`、`linux/arm64` 两种架构下的镜像构建、交付与部署规范。适用范围为 `app/`、`aiapp/`、`facade/`、`socketapp/` 下的 22 个服务。

## 约束

- 目标服务器无外网，镜像在构建机生成，以 `docker save` 导出的 tar 包交付，不依赖镜像仓库。
- 一个 tar 包只包含一个架构的镜像，不使用多架构 manifest list。
- 镜像架构必须与目标服务器一致；架构不匹配时容器启动报 `exec format error`。
- 未指定目标平台时默认 `amd64`（`deploy/common.sh` 中的 `DEFAULT_ARCH`），构建机自身架构不影响产物架构。

## 组成

| 文件 | 职责 |
| --- | --- |
| `deploy/common.sh` | `target_arch`：将 `linux/arm64`、`linux/arm64/v8`、`arm64` 统一解析为 `arm64`；`assert_image_arch`：校验镜像架构与目标一致 |
| `deploy/build-common.sh` | gis、file 使用的构建函数：准备基础镜像、`docker build`、架构校验、导出 tar |
| `deploy/base-image/` | runtime 基础镜像 `zero-service/runtime:<版本>-<arch>`，提供 GEOS、ffmpeg/ffprobe、CA 证书与时区，见 [`base-image/README.md`](./base-image/README.md) |
| `<服务目录>/Dockerfile` | 多阶段构建，构建上下文为仓库根目录 |
| `<服务目录>/build.sh` | 构建一个或多个架构的镜像并导出 tar，不访问远端 |
| `<服务目录>/deploy.sh` | 调用 `build.sh` 构建单一架构镜像，上传并在远端发布 |

## 服务分类

| 类型 | 服务 | 编译方式 | 运行镜像 | 跨架构构建 |
| --- | --- | --- | --- | --- |
| 纯 Go | 除 gis、file 外的 20 个服务 | `CGO_ENABLED=0`，交叉编译 | `scratch` | 不需要 qemu |
| 纯 Go + 外部命令 | file | `CGO_ENABLED=0`，交叉编译 | `zero-service/runtime` | 构建阶段不需要 qemu；基础镜像构建需要 |
| CGO | gis | `CGO_ENABLED=1`，链接 GEOS | `zero-service/runtime` | 需要 qemu/binfmt |

纯 Go 服务的 `scratch` 镜像只包含二进制、`etc/` 配置、系统 CA 证书和 `Asia/Shanghai` 时区文件。

## Dockerfile 规范

纯 Go 服务（含 file 的构建阶段）：

```dockerfile
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,id=go-build-${TARGETARCH},target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/<服务> ./<服务路径>

FROM scratch          # file 为 FROM ${BASE_IMAGE}
```

- 构建阶段固定运行在构建机架构（`$BUILDPLATFORM`），由 `TARGETOS`/`TARGETARCH` 决定产物架构。
- `ARG TARGETOS`/`TARGETARCH` 放在 `COPY . .` 之后，切换目标架构时复用依赖下载与源码层缓存。
- Go 编译缓存按 `TARGETARCH` 隔离，避免两种架构共享同一缓存目录。

CGO 服务（gis）：

```dockerfile
ARG BASE_IMAGE
FROM golang:1.26-alpine AS builder      # 不指定 $BUILDPLATFORM，按目标架构运行
ENV CGO_ENABLED=1
ARG GEOS_VERSION
RUN apk add --no-cache gcc musl-dev pkgconf geos-dev=${GEOS_VERSION}-r0
...
FROM ${BASE_IMAGE}
```

- 构建阶段与目标架构一致，跨架构时由 qemu 模拟执行。
- `geos-dev` 版本与基础镜像中的 GEOS 运行库版本均取自 `BASE_GEOS_VERSION`；builder 与基础镜像使用同一 Alpine 版本（当前 3.24），保证 `geos=<版本>-r0` 在两侧均可安装。
- `BASE_IMAGE` 不设默认值，由 `build.sh` 按目标架构传入，避免版本与架构在多处维护。

## build.sh

```bash
bash build.sh <环境> [平台列表]

bash build.sh test                              # 默认 linux/amd64 与 linux/arm64
bash build.sh test linux/arm64                  # 单一架构
bash build.sh test "linux/amd64 linux/arm64"    # 显式指定多个平台
```

- 读取 `env/<环境>.env`，仅要求 `IMAGE_NAME`。
- 每个平台依次执行：`docker build --platform linux/<arch>` → 架构校验 → `docker save` → 删除本地临时标签。
- 输出文件：`<IMAGE_NAME 中 / 替换为 ->_<arch>_<时间戳>.tar`，写入服务目录，已被 `.gitignore` 排除；镜像标签为 `<IMAGE_NAME>:<时间戳>-<arch>`。
- gis、file 在构建前通过 `deploy/base-image/ensure.sh` 确认对应架构的基础镜像，缺失时自动构建。

| 环境变量 | 说明 |
| --- | --- |
| `BUILD_TIMESTAMP` | 指定时间戳，默认取当前时间；`deploy.sh` 用它对齐本地与远端标签 |
| `BUILD_OUTPUT_ONLY` | 为 `true` 时 stdout 只输出 tar 文件名（每行一个），日志与构建输出写入 stderr |
| `HTTP_PROXY` / `HTTPS_PROXY` | 透传为构建参数 |
| `BASE_IMAGE` | 仅 gis、file：覆盖基础镜像，`ensure.sh` 只检查其存在性，不自动构建 |

## deploy.sh

```bash
bash deploy.sh <环境> [平台]

bash deploy.sh test_199                 # amd64
bash deploy.sh test_199 linux/arm64     # arm64
```

执行步骤：

1. 读取 `env/<环境>.env`，校验 `REMOTE_USER`、`REMOTE_PASSWD`、`REMOTE_HOST`、`REMOTE_PORT`、`REMOTE_PATH`、`REMOTE_COMPOSE_PATH`、`IMAGE_NAME`、`SERVICE_NAME`。
2. 读取远端 Docker daemon 架构（`docker version --format '{{.Server.Arch}}'`，不可用时回退 `uname -m`），与目标架构不一致时直接退出，不进入构建和上传。
3. 以 `BUILD_OUTPUT_ONLY=true` 调用 `build.sh` 构建目标架构镜像，获取 tar 文件名。
4. 通过 `scp` 上传 tar，失败重试 3 次。
5. 远端执行：`docker load` → 原 `<IMAGE_NAME>:<REMOTE_IMAGE_TAG>` 打 `backup_<时间戳>` 标签 → 新镜像打 `REMOTE_IMAGE_TAG` 标签 → `docker-compose up -d <SERVICE_NAME>` → 按 `BACKUP_KEEP` 清理旧备份 → 删除远端 tar。
6. 删除本地 tar。

| 可选变量 | 默认值 |
| --- | --- |
| `REMOTE_IMAGE_TAG` | `latest` |
| `BACKUP_KEEP` | `3` |

`deploy.sh` 只发布单一架构。向多台不同架构的服务器发布，按目标服务器分别指定平台执行；离线交付使用 `build.sh`。

## 离线交付

```bash
cd app/gis
bash build.sh prod
# zxhc.io-allcore-product-gis_amd64_<时间戳>.tar
# zxhc.io-allcore-product-gis_arm64_<时间戳>.tar
```

目标服务器按 `uname -m` 选择 tar 包（`x86_64` 对应 `amd64`，`aarch64` 对应 `arm64`）：

```bash
docker load -i <tar>
# compose 通过 ${REGISTER}/<服务>:${ALLCORE_MAIN_TAG} 引用镜像，标签需与之相同（通常为 1.0.0）
docker tag <IMAGE_NAME>:<时间戳>-<arch> <IMAGE_NAME>:<REMOTE_IMAGE_TAG>
docker-compose up -d <SERVICE_NAME>
```

## 基础镜像版本

基础镜像版本与 GEOS 版本在 `deploy/base-image/common.sh` 中维护：

| 变量 | 含义 |
| --- | --- |
| `BASE_RUNTIME_VERSION` | 基础镜像版本，体现在镜像标签中 |
| `BASE_GEOS_VERSION` | GEOS 版本，同时用于基础镜像与 gis 构建阶段 |

`ensure.sh` 仅按 `<版本>-<arch>` 判断镜像是否存在。修改基础镜像内容（包括 GEOS 版本、Alpine 版本、`deploy/tls/server.crt`）时必须同时提升 `BASE_RUNTIME_VERSION`，否则构建继续使用本地旧镜像。

## 构建机要求

- Docker 启用 BuildKit（Docker 23+ 默认启用）。
- 构建 gis 的非本机架构镜像或任一架构的基础镜像时，需要 qemu/binfmt。Docker Desktop 已内置；Linux 上的 Docker Engine 需执行：

  ```bash
  docker run --privileged --rm tonistiigi/binfmt --install all
  ```

## 故障排查

| 现象 | 原因 | 处理 |
| --- | --- | --- |
| `目标架构 'x' 与远端 ... 架构 'y' 不一致` | `deploy.sh` 的平台参数与服务器架构不符 | 按提示改用 `linux/<服务器架构>`，或确认连到了正确的服务器 |
| 容器启动报 `exec format error` | 镜像架构与服务器不一致 | `docker image inspect <镜像> -f '{{.Architecture}}'` 对比 `uname -m`，按正确平台重新构建 |
| gis 跨架构构建在 `apk add` 或 `go build` 处失败 | 未注册 qemu/binfmt | 按「构建机要求」安装 binfmt |
| `zero-service/runtime:<版本>-<arch>: not found` | 自定义 `BASE_IMAGE` 不存在，或基础镜像构建失败 | 执行 `bash deploy/base-image/ensure.sh <arch>` 查看具体错误 |
| `镜像架构 'x' 与目标架构 'y' 不一致` | 构建环境未按 `--platform` 产出镜像 | 确认 Docker 版本与 BuildKit 已启用 |
