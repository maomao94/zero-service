FROM golang:1.25-alpine3.22 AS builder

LABEL stage=gobuilder

ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG GOPROXY=https://goproxy.cn,direct

RUN set -x \
    && if [ -n "$HTTP_PROXY" ]; then export http_proxy=$HTTP_PROXY; fi \
    && if [ -n "$HTTPS_PROXY" ]; then export https_proxy=$HTTPS_PROXY; fi \
    && apk update --no-cache \
    && apk add --no-cache \
        tzdata make gcc libtool musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o gis ./app/gis && chmod +x gis

FROM golang:1.25-alpine3.22
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /usr/share/zoneinfo/Asia/Shanghai
ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /build/gis /app/gis
COPY --from=builder /build/app/gis/etc /app/etc

CMD ["./gis", "-f", "etc/gis.yaml"]