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

echo "生成完成，已移动至 ${OUT_DIR}/extproto"

protoc \
  --proto_path=. \
  --proto_path=${OUT_DIR} \
  --go_out=${OUT_DIR} \
  --go-grpc_out=${OUT_DIR} \
  dji_error_code.proto

mv ${OUT_DIR}/zero-service/third_party/dji_error_code ${OUT_DIR}/dji_error_code

echo "生成完成，已移动至 ${OUT_DIR}/dji_error_code"

# 删除多余目录
rm -rf ${OUT_DIR}/zero-service
echo "删除多余目录完成"