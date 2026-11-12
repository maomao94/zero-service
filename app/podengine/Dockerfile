# 业务镜像：纯 Go（CGO_ENABLED=0）多阶段构建，运行在 scratch。
# 由 app/podengine/build.sh 构建（deploy.sh 复用它），传 --platform，构建上下文为仓库根目录。
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS builder

LABEL stage=gobuilder

# 纯 Go 交叉编译：amd64 机器构建 arm64（或反向）都不需要 qemu
ENV CGO_ENABLED=0

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=$GOPROXY

# tzdata 用于给 scratch 运行镜像提供时区文件；ca-certificates 供 HTTPS 客户端使用
RUN set -x \
    && if [ -n "$HTTP_PROXY" ]; then export http_proxy=$HTTP_PROXY; fi \
    && if [ -n "$HTTPS_PROXY" ]; then export https_proxy=$HTTPS_PROXY; fi \
    && apk add --no-cache ca-certificates tzdata

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
    && GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/podengine ./app/podengine \
    && chmod +x /out/podengine

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /usr/share/zoneinfo/Asia/Shanghai
ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /out/podengine /app/podengine
COPY --from=builder /build/app/podengine/etc /app/etc

CMD ["./podengine", "-f", "etc/podengine.yaml"]
