#!/usr/bin/env bash

echo "开始生成"
OUT_DIR="../third_party"

# 清理旧生成
rm -rf ${OUT_DIR}/zero-service
rm -rf ${OUT_DIR}/extproto

# 执行生成
protoc \
  --proto_path=. \
  --proto_path=${OUT_DIR} \
  --go_out=${OUT_DIR} \
  --go-grpc_out=${OUT_DIR} \
  extproto.proto

# 移动到预期目录
mv ${OUT_DIR}/zero-service/third_party/extproto ${OUT_DIR}/extproto

# 删除多余目录
rm -rf ${OUT_DIR}/zero-service

echo "生成完成，已移动至 ${OUT_DIR}/extproto"