# 业务镜像：纯 Go（CGO_ENABLED=0）多阶段构建，运行在 deploy/base-image/Dockerfile 的 base 阶段提供的 base-runtime 基础镜像上。
# 由 app/opengtw/build.sh 构建（deploy.sh 复用它），传 BASE_IMAGE 与 --platform，构建上下文为仓库根目录。
# 纯 Go 服务的运行基础镜像 base-runtime 由 deploy/base-image/Dockerfile 提供（bash + 常用排查命令），
# BASE_IMAGE 由 build.sh 按目标架构传入；传 scratch 可构建无 shell 的精简镜像。
ARG BASE_IMAGE

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS builder

LABEL stage=gobuilder

# 纯 Go 交叉编译：amd64 机器构建 arm64（或反向）都不需要 qemu
ENV CGO_ENABLED=0

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=$GOPROXY

WORKDIR /build

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# TARGETOS/TARGETARCH 由 BuildKit 按 --platform 注入；放在 COPY 之后，切换架构时不失效已缓存的 COPY 层
ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,id=go-build-${TARGETARCH},target=/root/.cache/go-build \
    mkdir -p /out \
    && GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/opengtw ./app/opengtw \
    && chmod +x /out/opengtw

FROM ${BASE_IMAGE}

WORKDIR /app
COPY --from=builder /out/opengtw /app/opengtw
COPY --from=builder /build/app/opengtw/etc /app/etc

CMD ["./opengtw", "-f", "etc/opengtw.yaml"]
