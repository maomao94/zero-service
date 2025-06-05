FROM golang:alpine AS builder

LABEL stage=gobuilder

ENV CGO_ENABLED 0

# 接收代理参数（构建时通过 --build-arg 传递）
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG GOPROXY=https://goproxy.cn,direct

# 设置 Alpine 包管理器和 Go 的环境变量
RUN set -x \
    && if [ -n "$HTTP_PROXY" ]; then export http_proxy=$HTTP_PROXY; fi \
    && if [ -n "$HTTPS_PROXY" ]; then export https_proxy=$HTTPS_PROXY; fi \
    && apk update --no-cache \
    && apk add --no-cache tzdata


RUN apk update --no-cache && apk add --no-cache tzdata

WORKDIR /build

ADD go.mod .
ADD go.sum .

# 设置 Go Module 代理
RUN export GOPROXY=$GOPROXY \
    && go mod download
COPY . .

RUN go build -ldflags="-s -w" -o /app/xfusionmock app/xfusionmock/xfusionmock.go


FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /usr/share/zoneinfo/Asia/Shanghai
ENV TZ Asia/Shanghai

WORKDIR /app
COPY --from=builder /app/xfusionmock /app/xfusionmock
COPY app/xfusionmock/etc /app/etc

CMD ["./xfusionmock", "-f", "etc/xfusionmock.yaml"]
