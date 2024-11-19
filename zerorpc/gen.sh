#!/usr/bin/env bash

echo "开始生成"
goctl api format --dir=./
goctl rpc protoc zerorpc.proto --go_out=. --go-grpc_out=. --zrpc_out=. --client=true