# app/opengtw

`opengtw` 是开放平台的 HTTP 网关：签发应用凭据、校验 Basic 鉴权、统一响应信封，并把请求转发给 `app/openapi`。

## 服务信息

| 项 | 值 |
| --- | --- |
| HTTP 端口 | 11005 |
| 契约 | [`opengtw.api`](./opengtw.api) |
| 配置 | [`etc/opengtw.yaml`](./etc/opengtw.yaml) |
| 控制台 | `web/openapi`，生产路径 `/openapi/` |

## 对外接口

### 开放面

```http
POST /v1/gateway
GET  /v1/health
```

```json
{
  "method": "modbus.readHoldingRegisters",
  "data": {"modbusCode": "", "address": 0, "quantity": 2},
  "tid": "demo-1"
}
```

统一响应：

```json
{"code": 0, "msg": "ok", "data": {}, "timestamp": 1761120000000}
```

业务错误统一 HTTP 200，结果看 `code`；所有响应带 `X-Request-Id`。

应用签名为按应用配置的可选能力。`signRequired=false` 时沿用 Basic 鉴权；启用后请求 body 需要 `timestamp`、`nonce`、`sign`，签名规则与示例见 [`docs/open-api/signature.md`](../../docs/open-api/signature.md)。

### 管理面

```text
POST /open-admin/v1/app/create        创建应用，secret 仅返回一次
GET  /open-admin/v1/app/list          应用列表
POST /open-admin/v1/app/disable       启用/停用
POST /open-admin/v1/app/secret/reset  重置密钥，旧密钥保留 24h
```

管理面使用 JWT，默认密钥与 `gtw` / `livegtw` 一致。JWT 失败同样返回 `HTTP 200 + {"code":100401,"msg":"未认证"}`。nginx 默认不转发 `/open-admin/`。

## 鉴权

```http
Authorization: Basic base64(AppKey:AppSecret)
```

- 密钥只存 sha256，常数时间比较。
- 凭据缓存 60s；管理面变更后立即刷新。
- 停用应用立即返回 `130101`。

## 配置

- `DB`：应用凭据库；本地默认金仓 `zero`，dev/test 自动创建 `open_app`、`open_app_secret`。
- `OpenApiConf`：`app/openapi` 地址。
- `SignKeyEncryptionKey` 和 `Redis`：仅在应用启用 `signRequired` 时配置，用于保护签名 key 和 nonce 防重放。
- `JwtAuth.AccessSecret`：管理面 JWT 密钥。
- `MaxBytes`：请求体上限，默认 1 MiB。

## 本地启动

```bash
# 1. 转发服务（Mock）
cd app/openapi && go run . -f etc/openapi.yaml

# 2. HTTP 网关
cd app/opengtw && go run . -f etc/opengtw.yaml

# 3. 控制台
cd web/openapi && npm run dev
```

## 相关文档

- [开放平台文档](../../docs/open-api/README.md)
- [Modbus 开放接口](../../docs/open-api/modbus.md)
- [转发服务](../../app/openapi/README.md)
- [控制台](../../web/openapi/README.md)
