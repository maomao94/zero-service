# 业务镜像的两层运行基础镜像定义，由 deploy/build-image.sh 按 --target 构建：
#   base → zero-service/base-runtime：所有业务镜像的最小底座
#   env  → zero-service/env-runtime：base + GEOS + ffmpeg/ffprobe，供 gis、file 使用
# 构建上下文是 deploy/（见 build-image.sh），证书直接取 deploy/tls，保持与 nginx 一致。

FROM alpine:3.24 AS base

ARG HTTP_PROXY
ARG HTTPS_PROXY

# 运行期依赖一次装完：该层只在 Alpine 版本或工具清单变化时重建。
# bash 是硬要求：现场必须能 docker exec -it <容器> /bin/bash。
# curl/procps/iproute2/tcpdump：验证上游 HTTP 接口、查看进程与连接、抓包定位链路问题。
RUN set -x \
    && if [ -n "$HTTP_PROXY" ]; then export http_proxy=$HTTP_PROXY; fi \
    && if [ -n "$HTTPS_PROXY" ]; then export https_proxy=$HTTPS_PROXY; fi \
    && apk add --no-cache \
        bash ca-certificates tzdata curl procps iproute2 tcpdump \
    && test -x /bin/bash \
    && bash --version | head -n 1 \
    && test -s /etc/ssl/certs/ca-certificates.crt \
    && curl --version | head -n 1

# 内部自签证书单独一层：轮换 deploy/tls 只需重建这一层，不重新安装 apk 包；私钥不进入镜像。
# 所有业务镜像都信任与 nginx 相同的自签证书，调用内部 HTTPS 服务无需额外挂载。
COPY tls/server.crt /usr/local/share/ca-certificates/zero-service.crt
RUN update-ca-certificates \
    && grep -q "BEGIN CERTIFICATE" /usr/local/share/ca-certificates/zero-service.crt \
    && test -s /etc/ssl/certs/ca-certificates.crt

ENV TZ=Asia/Shanghai

WORKDIR /app

# GIS、file 的运行环境：在 base 之上叠加 C 依赖与媒体处理工具。
# 纯 Go 服务不需要这一层，直接用 base 以控制镜像体积。
FROM base AS env

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG GEOS_VERSION=3.14.1

RUN set -x \
    && if [ -n "$HTTP_PROXY" ]; then export http_proxy=$HTTP_PROXY; fi \
    && if [ -n "$HTTPS_PROXY" ]; then export https_proxy=$HTTPS_PROXY; fi \
    && apk add --no-cache \
        geos=${GEOS_VERSION}-r0 ffmpeg \
    && apk info -e geos \
    && ffmpeg -version | head -n 1 \
    && ffprobe -version | head -n 1
