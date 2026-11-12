# 业务镜像：Go 多阶段构建 + deploy/base-image 提供的 runtime 基础环境。
# 由 app/gis/build.sh 构建（经 deploy/build-common.sh 准备基础镜像，传 BASE_IMAGE、GEOS_VERSION 和 --platform）。
ARG BASE_IMAGE

FROM golang:1.26-alpine AS builder

LABEL stage=gobuilder

ENV GOOS=linux
# gis 通过 CGO 链接 GEOS，构建阶段需要 gcc 与 geos-dev（版本与基础镜像保持一致）
ENV CGO_ENABLED=1

ARG GEOS_VERSION=3.14.1
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=$GOPROXY

RUN set -x \
    && if [ -n "$HTTP_PROXY" ]; then export http_proxy=$HTTP_PROXY; fi \
    && if [ -n "$HTTPS_PROXY" ]; then export https_proxy=$HTTPS_PROXY; fi \
    && apk add --no-cache gcc musl-dev pkgconf geos-dev=${GEOS_VERSION}-r0 \
    && geos-config --version

WORKDIR /build

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# TARGETARCH 由 BuildKit 按 --platform 注入；放在 COPY 之后，切换架构时不失效已缓存的 COPY 层
ARG TARGETARCH

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,id=go-build-${TARGETARCH},target=/root/.cache/go-build \
    mkdir -p /out \
    && go build -trimpath -ldflags="-s -w" -o /out/gis ./app/gis \
    && chmod +x /out/gis

FROM ${BASE_IMAGE}

WORKDIR /app
COPY --from=builder /out/gis /app/gis
COPY --from=builder /build/app/gis/etc /app/etc

CMD ["./gis", "-f", "etc/gis.yaml"]
