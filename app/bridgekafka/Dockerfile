FROM golang:1.23-alpine3.22 AS builder

LABEL stage=gobuilder

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

ENV CGO_ENABLED 0

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG GOPROXY=https://goproxy.cn,direct

RUN set -x \
    && if [ -n "$HTTP_PROXY" ]; then export http_proxy=$HTTP_PROXY; fi \
    && if [ -n "$HTTPS_PROXY" ]; then export https_proxy=$HTTPS_PROXY; fi \
    && apk update --no-cache \
    && apk add --no-cache tzdata

RUN apk update --no-cache && apk add --no-cache tzdata

WORKDIR /build

COPY . .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /usr/share/zoneinfo/Asia/Shanghai
ENV TZ Asia/Shanghai

WORKDIR /app
COPY --from=builder /build /app/build
COPY etc /app/etc
COPY app/bridgekafka /app/bridgekafka

CMD ["./bridgekafka", "-f", "etc/bridgekafka.yaml"]
