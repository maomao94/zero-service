# 业务镜像：Go 多阶段构建 + deploy/base-image 提供的 runtime 基础环境。
# 由 app/file/build.sh 构建（经 deploy/build-common.sh 准备基础镜像，传 BASE_IMAGE 和 --platform）。
ARG BASE_IMAGE

# file 是纯 Go（CGO_ENABLED=0）：用构建机架构编译、按目标架构交叉编译，
# 因此 amd64 机器构建 arm64 镜像（或反向）都不需要 qemu 模拟，速度与本地构建一致。
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

LABEL stage=gobuilder

# ffmpeg/ffprobe 由运行期基础镜像提供，构建阶段不需要任何 apk 包
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
    && GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/file ./app/file \
    && chmod +x /out/file

FROM ${BASE_IMAGE}

WORKDIR /app
COPY --from=builder /out/file /app/file
COPY --from=builder /build/app/file/etc /app/etc

CMD ["./file", "-f", "etc/file.yaml"]
