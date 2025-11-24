FROM golang:1.25-alpine3.22 AS builder

LABEL stage=gobuilder

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 开启 cgo 并指定目标平台
ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG GOPROXY=https://goproxy.cn,direct

# 安装 gcc/musl-dev 和 tzdata
RUN set -x \
    && if [ -n "$HTTP_PROXY" ]; then export http_proxy=$HTTP_PROXY; fi \
    && if [ -n "$HTTPS_PROXY" ]; then export https_proxy=$HTTPS_PROXY; fi \
    && apk update --no-cache \
    && apk add --no-cache tzdata make gcc libtool musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o trigger ./app/trigger

# 最小 scratch 镜像
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /usr/share/zoneinfo/Asia/Shanghai
ENV TZ Asia/Shanghai

WORKDIR /app
COPY --from=builder /build/trigger /app/trigger
COPY etc /app/etc

CMD ["./trigger", "-f", "etc/trigger.yaml"]