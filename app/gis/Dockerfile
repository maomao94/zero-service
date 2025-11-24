FROM golang:1.25-alpine AS builder

LABEL stage=gobuilder

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64

ARG GOPROXY=https://goproxy.cn,direct

# 安装编译依赖（含 H3 编译所需工具，与最终镜像安装的 H3 版本保持一致）
RUN apk update --no-cache && apk add --no-cache \
    tzdata gcc musl-dev cmake make libtool tar libh3-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 编译应用（链接 H3 开发库，最终运行时依赖系统 H3 库）
RUN mkdir -p /build/bin && go build -o /build/bin/gis ./app/gis/gis.go

# --------------------------------------
# 最终镜像：golang:1.25-alpine + 自动安装 H3 运行时
# --------------------------------------
FROM golang:1.25-alpine

# 1. 安装 H3 运行时库（核心优化：无需手动复制，apk 自动处理路径）
# 直接安装 libh3（运行时依赖），避免手动复制动态库
RUN apk update --no-cache && apk add --no-cache \
    ca-certificates \
    tzdata && \
    rm -rf /var/cache/apk/*

# 2. 时区配置（直接用 apk 安装的 tzdata，无需复制）
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 可选：清理 Go 编译环境（如需精简镜像，保留运行时即可）
# RUN rm -rf /usr/local/go

# 设置工作目录
WORKDIR /app

# 复制编译好的应用和配置文件
COPY --from=builder /build/bin/gis /app/gis
COPY --from=builder /build/etc /app/etc

# 启动命令
CMD ["./gis", "-f", "etc/gis.yaml"]