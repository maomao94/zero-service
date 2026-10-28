#!/usr/bin/env bash
set -euo pipefail

# 自签 TLS 证书生成脚本
# 职责单一：只生成证书/私钥/SAN 配置，不改动部署配置、不修改系统信任链
#
# 用途：为 deploy/nginx 反向代理提供 TLS 证书（LiveKit 经反向代理访问，不单独配 TLS）
#
# 用法：
#   # 开发环境（默认）：localhost、回环、*.local、gitlab.localhost、livekit-server
#   ./gen-tls.sh
#
#   # 测试环境：必须指定主域名，SAN 只包含显式声明的入口
#   PROFILE=test DOMAIN=gitlab-test.example.com \
#   EXTRA_DNS="livekit-test.example.com" \
#   ./gen-tls.sh
#
#   # 生产环境：同上；生产建议改用受信任 CA 签发的证书，本脚本只负责自签
#   PROFILE=prod DOMAIN=gitlab.example.com \
#   EXTRA_DNS="livekit.example.com" \
#   EXTRA_IPS="203.0.113.10" \
#   ./gen-tls.sh
#
# 环境变量：
#   PROFILE      dev | test | prod，默认 dev
#   DOMAIN       主域名，同时作为 CN（dev 不填时 CN=localhost）
#   EXTRA_DNS    额外域名，逗号分隔
#   EXTRA_IPS    额外 IP，逗号分隔
#   LIVEKIT_DNS  LiveKit 入口主机名，dev 默认 livekit-server
#   CERT_DIR     输出目录，默认脚本所在目录
#   CERT_DAYS    有效期天数，默认 3650
#
# 生成物：
#   $CERT_DIR/server.crt   证书
#   $CERT_DIR/server.key   私钥
#   $CERT_DIR/san.cnf      证书签名请求用的 SAN 配置
#
# 信任问题：自签证书需由使用方自行决定是否信任（本脚本不做任何系统级操作）。
#
# SAN 与 CN 的职责（重要）：
#   - CN 只是证书的展示名，现代客户端不再用它做主机名校验
#   - 证书实际覆盖哪些地址完全由 subjectAltName（SAN）决定
#   - 客户端用哪个主机名连接，SAN 里就必须有同名条目（或匹配通配符），
#     否则会报 ERR_TLS_CERT_ALTNAME_INVALID，重试和信任都无法绕过
#   - 所以新增访问入口（域名或 IP）时，必须重签证书并 reload 网关，只改 nginx 不够
#
# 开发环境（PROFILE=dev）默认覆盖的入口（deploy/nginx 各 vhost 共用一张证书）：
#   localhost / 127.0.0.1 / ::1   本机与回环访问
#   *.local                       本地子域名，例如 gitlab.local
#   gitlab.localhost              GitLab 反向代理入口（OpenChamber 实例与 git 远端共用）
#   livekit-server                LiveKit 经网关反向代理访问时使用的主机名
#
# 上测试 / 生产环境时需要同步修改的配置（按顺序检查）：
#   1. DNS          把 gitlab.<域名>、livekit.<域名> 解析到网关（测试/生产不要依赖 hosts）
#   2. GitLab       external_url 改为对外域名，否则 API 返回的 clone/ssh 地址仍是旧主机，
#                   OpenChamber 的实例匹配会失败
#   3. nginx        deploy/nginx/gitlab.conf 的 server_name 改为对外域名，重新部署或 reload
#   4. 证书         用 PROFILE=test/prod 重签；生产优先使用受信任 CA 签发的证书
#   5. 信任         自签证书需要客户端导入（浏览器、Node 的 NODE_EXTRA_CA_CERTS 等）
#   6. OpenChamber  实例地址改为 https://gitlab.<域名> 并重连 PAT；git 远端使用
#                   scp 风格 git@gitlab.<域名>:<group>/<repo>.git，SSH 端口写在 ~/.ssh/config
#   7. 端口         对外不再使用 8929 / 2224 时，同步修改 nginx upstream、ssh config 和远端地址

CERT_DIR="${CERT_DIR:-$(cd "$(dirname "$0")" && pwd)}"
DAYS="${CERT_DAYS:-3650}"
PROFILE="${PROFILE:-dev}"
DOMAIN="${DOMAIN:-}"
LIVEKIT_DNS="${LIVEKIT_DNS:-}"

case "$PROFILE" in
  dev|test|prod) ;;
  *) echo "错误：PROFILE 只能是 dev / test / prod" >&2; exit 2 ;;
esac

if [ "$PROFILE" != "dev" ] && [ -z "$DOMAIN" ]; then
  echo "错误：PROFILE=$PROFILE 时必须指定 DOMAIN（对外主域名）" >&2
  exit 2
fi

CN="${DOMAIN:-localhost}"

# SAN 条目去重累积（bash 3.2 无关联数组）
DNS_LIST=""
IP_LIST=""

add_dns() {
  [ -n "${1:-}" ] || return 0
  case " $DNS_LIST " in *" $1 "*) return 0 ;; esac
  DNS_LIST="${DNS_LIST}${DNS_LIST:+ }$1"
}

add_ip() {
  [ -n "${1:-}" ] || return 0
  case " $IP_LIST " in *" $1 "*) return 0 ;; esac
  IP_LIST="${IP_LIST}${IP_LIST:+ }$1"
}

# 开发环境带入本地入口；test/prod 只用显式声明的域名/IP，避免把回环地址写进线上证书
if [ "$PROFILE" = "dev" ]; then
  add_dns localhost
  add_ip 127.0.0.1
  add_ip ::1
  add_dns '*.local'
  add_dns gitlab.localhost
  add_dns "${LIVEKIT_DNS:-livekit-server}"
fi

[ -n "$DOMAIN" ] && add_dns "$DOMAIN"
[ -n "$LIVEKIT_DNS" ] && add_dns "$LIVEKIT_DNS"

IFS=',' read -ra EXTRA_DNS_ARR <<< "${EXTRA_DNS:-}"
for d in ${EXTRA_DNS_ARR[@]+"${EXTRA_DNS_ARR[@]}"}; do
  add_dns "$(echo "$d" | xargs)"
done

IFS=',' read -ra EXTRA_IPS_ARR <<< "${EXTRA_IPS:-}"
for ip in ${EXTRA_IPS_ARR[@]+"${EXTRA_IPS_ARR[@]}"}; do
  add_ip "$(echo "$ip" | xargs)"
done

if [ -z "$DNS_LIST" ] && [ -z "$IP_LIST" ]; then
  echo "错误：没有可写入 SAN 的域名或 IP，请检查 DOMAIN / EXTRA_DNS / EXTRA_IPS" >&2
  exit 2
fi

echo "=== 生成自签 TLS 证书 ==="
echo "环境: $PROFILE"
echo "证书目录: $CERT_DIR"
echo "CN / 主域名: $CN"
echo "有效期: ${DAYS} 天"
echo ""

mkdir -p "$CERT_DIR"

openssl genrsa -out "$CERT_DIR/server.key" 2048 2>/dev/null
# 私钥只允许当前用户读取
chmod 600 "$CERT_DIR/server.key"

# 生成 SAN 配置（含环境与用途注释）
{
cat <<EOF
# 本文件由 deploy/tls/gen-tls.sh 生成，请勿手工修改
# 环境: $PROFILE
# CN 仅作展示；证书覆盖范围以下面的 subjectAltName 为准
# 新增入口后需重新生成证书并 reload 网关

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
EOF

dns_idx=1
for d in $DNS_LIST; do
  echo "DNS.$dns_idx = $d"
  dns_idx=$((dns_idx + 1))
done

ip_idx=1
for ip in $IP_LIST; do
  echo "IP.$ip_idx = $ip"
  ip_idx=$((ip_idx + 1))
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
echo "覆盖域名: ${DNS_LIST:-（无）}"
echo "覆盖 IP  : ${IP_LIST:-（无）}"
echo "新增入口需重新执行本脚本并 reload 网关，详见文件头注释。"
echo ""

if [ "$PROFILE" != "dev" ]; then
  echo "⚠️  $PROFILE 环境提醒："
  echo "   - 确认 GitLab external_url、nginx server_name 已改为对外域名"
  echo "   - 生产环境优先使用受信任 CA 签发的证书，而不是自签"
  echo "   - OpenChamber 实例地址需改为 https://gitlab.<域名> 并重新连接 PAT"
  echo ""
fi

echo "=== 配置完成（未改动任何部署文件、未修改系统信任链）==="
echo ""
echo "下一步："
echo "  - 由 deploy/nginx 反向代理使用该证书（HTTPS 终止），LiveKit 保持内部 ws，不单独配 TLS"
echo "  - 部署网关: bash deploy/nginx/deploy.sh"
echo "  - 自签证书的信任由你自行决定（浏览器手动信任或导入系统，本脚本不代劳）"
