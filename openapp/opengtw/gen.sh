#!/usr/bin/env bash

echo "开始生成"
goctl api format --dir=./
goctl api go --api=./opengtw.api --dir=./

# goctl api 语法不支持 json.RawMessage（点号会报语法错误），
# 生成后把 data 字段还原为 json.RawMessage，保证网关原样透传、不做二次序列化。
TYPES_FILE=internal/types/types.go
if ! grep -q 'json.RawMessage' "$TYPES_FILE"; then
  perl -0pi -e 's/Data(\s+)interface\{\}/Data${1}json.RawMessage/g' "$TYPES_FILE"
  perl -0pi -e 's/package types\n/package types\n\nimport "encoding\/json"\n/' "$TYPES_FILE"
fi
