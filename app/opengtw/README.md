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

应用签名为按应用配置的可选能力。`signRequired=false` 时沿用 Basic 鉴权；启用后请求需要携带 `X-Sign: v1:<hex>` 头，`timestamp`/`nonce` 放在 body 中，签名覆盖原始 body 字节，`data` 保持 JSON 对象。规则与示例见 [`docs/open-api/signature.md`](../../docs/open-api/signature.md)。

### 管理面

```text
POST /open-admin/v1/app/create        创建应用，secret 仅返回一次
GET  /open-admin/v1/app/list          应用列表
POST /open-admin/v1/app/disable       启用/停用
POST /open-admin/v1/app/secret/reset  重置密钥，旧密钥保留 24h
POST /open-admin/v1/app/sign-key/reveal  查看当前 signKey（受配置开关控制，记录审计日志）
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
- `SignKeyRevealEnabled`：是否允许管理面查看当前 signKey，默认 `false`；开启后建议只在内网暴露 `/open-admin/`。
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

## 本地 keytool

`cmd/keytool` 直接读配置和凭据库，不依赖 HTTP 管理面、Redis 或 gRPC：

```bash
# 查看应用当前生效的 signKey（key 只写 stdout）
go run ./app/opengtw/cmd/keytool reveal -app app_xxx

# 生成签名 body、canonical、X-Sign 和 curl
go run ./app/opengtw/cmd/keytool sign \
  -app app_xxx -method pod.list -data '{"limit":10,"offset":0}' \
  -secret sec_xxx

# 签名后直接调用网关，形成本地闭环
go run ./app/opengtw/cmd/keytool sign \
  -app app_xxx -method pod.list -data '{"limit":10}' \
  -secret sec_xxx -send
```

默认读取 `app/opengtw/etc/opengtw.yaml`；在其他目录执行时用 `-f` 指定。工具需要能读到 `SignKeyEncryptionKey`，否则只能拿到“签名密钥未配置”的错误。`signKey` 是敏感凭据，工具只把 key 写到 stdout，提示和错误写到 stderr。

> keytool 直接读配置和凭据库，不受管理面 `SignKeyRevealEnabled` 开关限制；能拿到配置和数据库访问权限的人本来就能解密。开关只用于约束 HTTP 管理面。

## 相关文档

- [开放平台文档](../../docs/open-api/README.md)
- [Modbus 开放接口](../../docs/open-api/modbus.md)
- [转发服务](../../app/openapi/README.md)
- [控制台](../../web/openapi/README.md)
