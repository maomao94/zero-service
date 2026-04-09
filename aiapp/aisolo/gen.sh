#!/usr/bin/env bash

echo "开始生成 aisolo proto"
goctl rpc protoc aisolo.proto --go_out=. --go-grpc_out=. --zrpc_out=. --client=false
