# 纯 Go 业务服务的运行基础镜像：bash + 常用排查命令 + CA 证书/时区。
# 由 deploy/base-image/build.sh go 构建，业务 Dockerfile 只叠加二进制和配置。
# 构建上下文是 deploy/，本文件不引用上下文内的其它文件。

FROM alpine:3.24

ARG HTTP_PROXY
ARG HTTPS_PROXY

# bash：现场直接 docker exec -it <容器> /bin/bash
# curl/procps/iproute2/tcpdump：验证上游 HTTP 接口、查看进程与连接、抓包定位链路问题
RUN set -x \
    && if [ -n "$HTTP_PROXY" ]; then export http_proxy=$HTTP_PROXY; fi \
    && if [ -n "$HTTPS_PROXY" ]; then export https_proxy=$HTTPS_PROXY; fi \
    && apk add --no-cache \
        bash ca-certificates tzdata curl procps iproute2 tcpdump \
    && test -x /bin/bash \
    && test -s /etc/ssl/certs/ca-certificates.crt \
    && curl --version | head -n 1

ENV TZ=Asia/Shanghai

WORKDIR /app
