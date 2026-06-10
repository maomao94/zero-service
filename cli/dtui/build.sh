#!/usr/bin/env bash
set -euo pipefail

# dtui 编译脚本
# 用法: cd cli/dtui && ./build.sh
# 产物输出到 cli/dtui/bin/

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
OUT_DIR="$SCRIPT_DIR/bin"

mkdir -p "$OUT_DIR"

echo "=== 编译 dtui ==="
echo "来源: $SCRIPT_DIR"
echo "产物: $OUT_DIR"
echo ""

# 清理旧产物
rm -f "$OUT_DIR"/dtui*

# 本机平台
echo "编译: bin/dtui (本机平台)"
(cd "$REPO_ROOT" && go build -o "$OUT_DIR/dtui" ./cli/dtui)

# 交叉编译: Darwin + Linux（amd64 / arm64）
for target in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64; do
	os="${target%/*}"
	arch="${target#*/}"
	out="$OUT_DIR/dtui-${os}-${arch}"
	echo "编译: $(basename "$out")"
	(cd "$REPO_ROOT" && GOOS="$os" GOARCH="$arch" go build -o "$out" ./cli/dtui)
done

echo ""
echo "编译完成。产物列表："
ls -la "$OUT_DIR"/dtui*
