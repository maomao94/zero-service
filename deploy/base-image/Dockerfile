# gis/file 等服务共用的运行基础镜像：GEOS 运行库 + ffmpeg/ffprobe + 内部自签证书。
# 由 deploy/base-image/build.sh 构建，业务 Dockerfile 只叠加二进制和配置。
# 构建上下文是 deploy/（见 build.sh），证书直接取 deploy/tls，保持与 nginx 一致。

FROM alpine:3.24

ARG GEOS_VERSION=3.14.1
ARG HTTP_PROXY
ARG HTTPS_PROXY

# 运行期依赖一次装完：该层只在 GEOS/ffmpeg/Alpine 版本变化时重建
RUN set -x \
    && if [ -n "$HTTP_PROXY" ]; then export http_proxy=$HTTP_PROXY; fi \
    && if [ -n "$HTTPS_PROXY" ]; then export https_proxy=$HTTPS_PROXY; fi \
    && apk add --no-cache \
        geos=${GEOS_VERSION}-r0 ca-certificates tzdata ffmpeg \
    && apk info -e geos \
    && test -s /etc/ssl/certs/ca-certificates.crt \
    && ffmpeg -version | head -n 1

# 证书单独一层：轮换 deploy/tls 只需重建这一层，不重新安装 apk 包；私钥不进入镜像
COPY tls/server.crt /usr/local/share/ca-certificates/zero-service.crt
RUN update-ca-certificates \
    && grep -q "BEGIN CERTIFICATE" /usr/local/share/ca-certificates/zero-service.crt \
    && test -s /etc/ssl/certs/ca-certificates.crt

ENV TZ=Asia/Shanghai

WORKDIR /app
