#!/usr/bin/env bash
# 前端构建打包脚本
# 构建各 web 子项目，并把产物汇总到 deploy/nginx/dist/<app>/
# nginx 只读挂载该 dist 目录（未来测试/生产可直接覆盖或上传此目录）
#
# 用法:
#   bash deploy/nginx/build-web.sh              # 构建全部（live/socketio/workspace）
#   bash deploy/nginx/build-web.sh live         # 只构建 live
#   bash deploy/nginx/build-web.sh live socketio

set -euo pipefail
cd "$(dirname "$0")"
NGINX_DIR="$(pwd)"
ROOT="$(cd ../.. && pwd)"
DIST_DIR="$NGINX_DIR/dist"

APPS=("$@")
if [ ${#APPS[@]} -eq 0 ]; then
  APPS=(live socketio workspace)
fi

for app in "${APPS[@]}"; do
  src="$ROOT/web/$app"
  if [ ! -d "$src" ]; then
    echo "跳过：web/$app 不存在"
    continue
  fi
  echo ""
  echo "=== 构建 web/$app ==="
  ( cd "$src" && npm install --no-audit --no-fund && npm run build )
  dest="$DIST_DIR/$app"
  # 只清空目录内容，不删除目录本身：dist/<app> 被 nginx 容器只读挂载，
  # 删除重建会让 bind mount 指向旧 inode，容器内看到的是空目录（403）
  mkdir -p "$dest"
  find "$dest" -mindepth 1 -maxdepth 1 -exec rm -rf {} +
  cp -R "$src/dist/." "$dest/"
  echo "✅ web/$app -> deploy/nginx/dist/$app"
done

echo ""
echo "前端产物目录（供 nginx 挂载）: $DIST_DIR"
ls -1 "$DIST_DIR" 2>/dev/null || true
echo ""
echo "下一步: bash deploy/nginx/deploy.sh"
