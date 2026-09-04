#!/usr/bin/env bash
set -euo pipefail

# 自签 TLS 证书生成脚本
# 测试/生产环境通用：通过环境变量注入域名和 IP，不绑死任何地址
#
# 用法：
#   # 开发环境（默认 localhost + loopback）
#   ./gen-tls.sh
#
#   # 测试/生产环境（注入域名 + 额外 IP/域名）
#   DOMAIN=meet.example.com \
#   EXTRA_DNS="api.meet.example.com,*.meet.example.com" \
#   EXTRA_IPS="10.0.0.1,172.16.0.1" \
#   ./gen-tls.sh

CERT_DIR="${CERT_DIR:-$(cd "$(dirname "$0")" && pwd)}"
DAYS="${CERT_DAYS:-3650}"
CN="${DOMAIN:-localhost}"

# 解析额外 SAN 条目（逗号分隔）
IFS=',' read -ra EXTRA_DNS_ARR <<< "${EXTRA_DNS:-}"
IFS=',' read -ra EXTRA_IPS_ARR <<< "${EXTRA_IPS:-}"

echo "=== 生成自签 TLS 证书 ==="
echo "证书目录: $CERT_DIR"
echo "CN / 主域名: $CN"
echo "有效期: ${DAYS} 天"
echo ""

mkdir -p "$CERT_DIR"

openssl genrsa -out "$CERT_DIR/server.key" 2048 2>/dev/null

# 动态生成 SAN 配置
{
cat <<EOF
[req]
default_bits = 2048
prompt = no
default_md = sha256
distinguished_name = dn
x509_extensions = v3_req

[dn]
C = CN
ST = Beijing
L = Beijing
O = Zero Service
OU = Dev
CN = $CN

[v3_req]
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.local
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

dns_idx=3
for d in ${EXTRA_DNS_ARR[@]+"${EXTRA_DNS_ARR[@]}"}; do
  d=$(echo "$d" | xargs)
  [ -z "$d" ] && continue
  echo "DNS.$dns_idx = $d"
  ((dns_idx++))
done

ip_idx=3
for ip in ${EXTRA_IPS_ARR[@]+"${EXTRA_IPS_ARR[@]}"}; do
  ip=$(echo "$ip" | xargs)
  [ -z "$ip" ] && continue
  echo "IP.$ip_idx = $ip"
  ((ip_idx++))
done
} > "$CERT_DIR/san.cnf"

openssl req -x509 -nodes -days "$DAYS" \
  -key "$CERT_DIR/server.key" \
  -out "$CERT_DIR/server.crt" \
  -config "$CERT_DIR/san.cnf" 2>/dev/null

echo "✅ 证书已生成:"
echo "   $CERT_DIR/server.crt"
echo "   $CERT_DIR/server.key"
echo ""

echo "=== 证书详情 ==="
openssl x509 -in "$CERT_DIR/server.crt" -noout -subject -dates -ext subjectAltName 2>/dev/null
echo ""

# 更新 livekit-server.yaml（添加 TLS 配置）
LK_YAML="$CERT_DIR/../livekit/livekit-server.yaml"
if [ -f "$LK_YAML" ]; then
  if ! grep -q "^tls:" "$LK_YAML"; then
    echo "=== 更新 livekit-server.yaml（添加 TLS） ==="
    cat >> "$LK_YAML" <<EOF

# TLS 配置（自签证书，支持 WSS）
tls:
  cert_file: /etc/livekit/tls/server.crt
  key_file: /etc/livekit/tls/server.key
EOF
    echo "✅ 已添加 TLS 配置到 livekit-server.yaml"
  else
    echo "⏭️  livekit-server.yaml 已有 TLS 配置，跳过"
  fi
fi

# 更新 docker-compose.yaml（挂载证书）
DC_YAML="$CERT_DIR/../livekit/docker-compose.yaml"
if [ -f "$DC_YAML" ] && ! grep -q "tls/server.crt" "$DC_YAML"; then
  echo "=== 更新 docker-compose.yaml（挂载证书） ==="
  sed -i.bak \
    's|./livekit-server.yaml:/etc/livekit.yaml|./livekit-server.yaml:/etc/livekit.yaml\n      - ../tls/server.crt:/etc/livekit/tls/server.crt:ro\n      - ../tls/server.key:/etc/livekit/tls/server.key:ro|' \
    "$DC_YAML" 2>/dev/null || true
  echo "✅ 已更新 docker-compose.yaml"
fi

# 安装到系统信任链（macOS / Linux）
echo ""
echo "=== 安装证书到系统信任链 ==="
case "$(uname -s)" in
  Darwin)
    if security verify-cert -c "$CERT_DIR/server.crt" >/dev/null 2>&1; then
      echo "⏭️  证书已在系统信任链中，跳过"
    else
      sudo security add-trusted-cert -d -r trustRoot \
        -k /Library/Keychains/System.keychain "$CERT_DIR/server.crt" \
        && echo "✅ 已加入 macOS 系统信任链"
    fi
    ;;
  Linux)
    echo "请手动执行（需 root）："
    echo "  cp $CERT_DIR/server.crt /usr/local/share/ca-certificates/zero-dev.crt"
    echo "  update-ca-certificates"
    ;;
esac
echo ""

echo "=== 配置完成 ==="
echo ""
echo "SAN 条目:"
openssl x509 -in "$CERT_DIR/server.crt" -noout -ext subjectAltName 2>/dev/null | sed 's/    //'
echo ""
echo "下一步："
echo "1. docker compose restart livekit-server"
echo "2. 服务端 LiveKit.Url: https://$CN:7880"
echo "3. 前端 vite 自动检测 deploy/tls/server.crt 启用 HTTPS"
echo ""
echo "生产环境用法:"
echo "  DOMAIN=meet.yourcompany.com \\"
echo "  EXTRA_DNS=\"api.meet.yourcompany.com\" \\"
echo "  EXTRA_IPS=\"\$(curl -s ifconfig.me)\" \\"
echo "  ./gen-tls.sh"
