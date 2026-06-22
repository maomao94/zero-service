FROM golang:1.26-alpine3.22 AS builder

LABEL stage=gobuilder

ENV TZ=Asia/Shanghai

ENV CGO_ENABLED=1
ENV GOOS=linux

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=$GOPROXY

RUN set -x \
    && if [ -n "$HTTP_PROXY" ]; then export http_proxy=$HTTP_PROXY; fi \
    && if [ -n "$HTTPS_PROXY" ]; then export https_proxy=$HTTPS_PROXY; fi \
    && apk add --no-cache \
        tzdata ca-certificates make gcc musl-dev pkgconf geos-dev \
    && ln -snf /usr/share/zoneinfo/$TZ /etc/localtime \
    && echo $TZ > /etc/timezone \
    && geos-config --version

WORKDIR /build

ARG TARGETARCH

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,id=go-build-${TARGETARCH},target=/root/.cache/go-build \
    mkdir -p /out \
    && go build -trimpath -ldflags="-s -w" -o /out/gis ./app/gis \
    && chmod +x /out/gis

FROM alpine:3.22
RUN apk add --no-cache geos \
    && mkdir -p /etc/ssl/certs /usr/share/zoneinfo/Asia
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo/Asia/Shanghai /usr/share/zoneinfo/Asia/Shanghai
ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /out/gis /app/gis
COPY --from=builder /build/app/gis/etc /app/etc

CMD ["./gis", "-f", "etc/gis.yaml"]
