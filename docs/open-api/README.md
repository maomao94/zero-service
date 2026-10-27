# app 开放 API 网关（Podman + Modbus）

`opengtw` 面向第三方应用（含 Java 服务端）提供统一的 HTTP/JSON 开放 API，当前接入 Podman 容器能力与 Modbus 读写能力。网关负责 Basic 鉴权、统一信封、错误码折叠和请求日志；实际 method 路由与 protojson 转换由 `app/openapi` 完成。

## 架构与端口

```text
第三方 / Java
   │  HTTP + Basic                    │  gRPC（内网直连，不做鉴权）
   ▼                                  ▼
app/opengtw :11005  ──gRPC──▶  app/openapi :21018  ──gRPC──▶  app/podengine :21010
（鉴权、信封、管理接口）           （method 路由、protojson、Mock）        └──gRPC──▶  app/bridgemodbus :25004
```

| 服务 | 目录 | 端口 | 说明 |
| --- | --- | --- | --- |
| `opengtw` | `app/opengtw` | 11005 | HTTP 网关，`/v1/*` 与 `/open-admin/v1/*` |
| `openapi` | `app/openapi` | 21018 | gRPC 转发服务，`OpenApiRpc.Invoke` |
| `podengine` | `app/podengine` | 21010 | 下游容器管理服务 |
| `bridgemodbus` | `app/bridgemodbus` | 25004 | 下游 Modbus TCP 读写服务 |

## 测试控制台

`web/openapi` 是配套的浏览器调试台，仅限开发/测试环境：

- 开发端口 `5181`，生产挂在 `/openapi/`。
- API 调试：19 个 method（Podman 4 + Modbus 15）、JSON 编辑器、写操作提示、响应高亮、`X-Request-Id`/耗时/大小/错误码展示。
- 应用管理：JWT 鉴权，创建、启停、重置密钥，secret 只展示一次。
- 请求历史：最近 50 条，支持过滤与回填；不保存 secret。
- 错误码速查：BC=30 与常用 extproto 错误码。

开发态 Vite 会把 `/v1` 和 `/open-admin` 代理到 `127.0.0.1:11005`；生产态由 nginx 反代 `/v1/`，默认不转发 `/open-admin/`。详见 [`web/openapi/README.md`](../../web/openapi/README.md)。

## 鉴权

```http
Authorization: Basic base64(appId:secret)
```

- `appId` 形如 `app_` + 32 位 UUID hex；`secret` 与 `signKey` 都是随机密钥，创建/重置时返回一次；`signKey` 还可在管理面按开关查看，`secret` 无法找回。
- 服务端只保存 `sha256(secret)` 的十六进制摘要，并使用常数时间比较。
- 应用不存在或已停用返回 `130101`；密钥缺失或不匹配返回 `130102`。HTTP 状态统一为 200，业务结果以响应体 `code` 为准。
- 凭据缓存在内存中，每 60s 从凭据库刷新；管理面变更后立即刷新。本地默认使用金仓 `zero` 库，生产建议独立 `resource` 库。
- 每个应用每分钟最多更新一次 `last_used_at`，不会逐请求查库。

## 统一转发

```http
POST /v1/gateway
Content-Type: application/json
Authorization: Basic ...
```

请求：

```json
{
  "method": "pod.list",
  "data": {"node": "local", "limit": 100, "offset": 0},
  "tid": "req-20261022-0001"
}
```

应用的 `signRequired` 为 `true` 时，请求必须携带 `X-Sign: v1:<hex>`，并把 `timestamp`、`nonce` 放进 body：

```http
POST /v1/gateway
Authorization: Basic ...
X-Sign: v1:a21688f6f7bd082ead7fa5b1a34eb48899210c7702d808d6413e3197cbea13ba
Content-Type: application/json

{"method":"pod.list","data":{"limit":100,"offset":0},"tid":"req-001","timestamp":1761120000000,"nonce":"nonce-001"}
```

签名 canonical string 固定为 4 行（换行分隔）：

```text
v1
{appId}
{path}
{bodyHash}
```

签名使用 `HMAC-SHA256`；`bodyHash = SHA256(原始 body 字节)`，`timestamp`/`nonce` 也由 bodyHash 覆盖。`data` 保持 JSON 对象，不需要改成字符串。`timestamp` 允许与服务端相差 5 分钟，`nonce` 由 Redis 短期占用以防重放。未开启签名的应用继续使用现有 Basic 鉴权，不需要 `X-Sign`。

密钥格式、`secret`/`signKey` 的密码学处理、canonical string 详解和完整示例见 [`signature.md`](./signature.md)。

响应：

```json
{"code": 0, "msg": "ok", "data": {"items": [], "total": 0}, "timestamp": 1761120000000}
```

- `tid` 会回显到响应头 `X-Request-Id`；未传时由网关生成。所有响应（包括鉴权失败、解析失败）都会带 `X-Request-Id`，并优先复用请求头里的同名值。
- 所有响应统一为 HTTP 200，响应体固定为 `{code,msg,data,timestamp}`；错误码表里的 HTTP 列表示语义状态，不代表实际响应状态。
- `data` 为该 method 的 protojson，网关按原始字节转发，不做二次序列化。

### 可用 method

#### Podman

| method | 下游 RPC | 说明 |
| --- | --- | --- |
| `pod.list` | `podengine.PodEngine/ListPods` | 容器列表，支持名称/ID/标签、分页 |
| `pod.get` | `podengine.PodEngine/GetPod` | 容器详情 |
| `pod.stats` | `podengine.PodEngine/GetPodStats` | 容器资源统计 |
| `image.list` | `podengine.PodEngine/ListImages` | 镜像列表 |

#### Modbus

| method | 下游 RPC | 功能码/说明 |
| --- | --- | --- |
| `modbus.readCoils` | `bridgemodbus.BridgeModbus/ReadCoils` | 0x01 读线圈 |
| `modbus.readDiscreteInputs` | `bridgemodbus.BridgeModbus/ReadDiscreteInputs` | 0x02 读离散输入 |
| `modbus.readInputRegisters` | `bridgemodbus.BridgeModbus/ReadInputRegisters` | 0x04 读输入寄存器 |
| `modbus.readHoldingRegisters` | `bridgemodbus.BridgeModbus/ReadHoldingRegisters` | 0x03 读保持寄存器 |
| `modbus.readFIFOQueue` | `bridgemodbus.BridgeModbus/ReadFIFOQueue` | 0x18 读 FIFO 队列 |
| `modbus.readDeviceIdentification` | `bridgemodbus.BridgeModbus/ReadDeviceIdentification` | 0x2B/0x0E 读设备标识 |
| `modbus.readDeviceIdentificationSpecificObject` | `bridgemodbus.BridgeModbus/ReadDeviceIdentificationSpecificObject` | 读指定设备对象 |
| `modbus.writeSingleCoil` | `bridgemodbus.BridgeModbus/WriteSingleCoil` | 0x05 写单个线圈 |
| `modbus.writeMultipleCoils` | `bridgemodbus.BridgeModbus/WriteMultipleCoils` | 0x0F 写多个线圈 |
| `modbus.writeSingleRegister` | `bridgemodbus.BridgeModbus/WriteSingleRegister` | 0x06 写单个寄存器 |
| `modbus.writeSingleRegisterWithDecimal` | `bridgemodbus.BridgeModbus/WriteSingleRegisterWithDecimal` | 十进制写单个寄存器 |
| `modbus.writeMultipleRegisters` | `bridgemodbus.BridgeModbus/WriteMultipleRegisters` | 0x10 写多个寄存器 |
| `modbus.writeMultipleRegistersWithDecimal` | `bridgemodbus.BridgeModbus/WriteMultipleRegistersWithDecimal` | 十进制写多个寄存器 |
| `modbus.readWriteMultipleRegisters` | `bridgemodbus.BridgeModbus/ReadWriteMultipleRegisters` | 0x17 读写多个寄存器 |
| `modbus.maskWriteRegister` | `bridgemodbus.BridgeModbus/MaskWriteRegister` | 0x16 掩码写寄存器 |

Modbus 详细字段、示例和响应说明见 [`modbus.md`](./modbus.md)。未注册的 method 返回 `130103`。

### 请求示例

```bash
curl -sS -X POST http://127.0.0.1:11005/v1/gateway \
  -H 'Content-Type: application/json' \
  -u 'app_0123456789ab:sec_0123456789abcdef0123456789abcdef' \
  -d '{
    "method": "pod.list",
    "data": {"node": "local", "limit": 20, "offset": 0},
    "tid": "demo-1"
  }'
```

### 存活探针

```http
GET /v1/health
```

不需要鉴权，返回 `{"code":0,"msg":"ok","data":{"status":"ok"}}`。

## protojson 注意事项

- 字段使用 protojson 的 camelCase 名称，例如 `node`、`limit`、`removeVolumes`。
- `int64` / `uint64` 在输出中固定为字符串；输入时也接受数字。
- 枚举使用名称，例如 `POD_PHASE_RUNNING`。
- `bytes` 字段输出为 base64 字符串，例如 Modbus 的 `results`。
- 未知字段会被拒绝，返回 `100101`（HTTP 200 + 业务码）。
- 按 protojson 规范，32 位整型同时接受数字和数字字符串，例如 `{"limit":"100"}` 是合法的。需要排错时以 `app/openapi/openapi.proto` 与 podengine proto 为准。

## 错误码

| code | 语义 HTTP | 含义 |
| --- | --- | --- |
| `0` | 200 | 成功 |
| `100101` | 400 | 参数错误 / data 不是合法 protojson |
| `100102` | 400 | 缺少必要参数 |
| `100202` | 404 | 记录不存在 |
| `100401` | 401 | 未认证 |
| `100402` | 403 | 无权限访问 |
| `100701` | 429 | 请求过于频繁 |
| `100502` | 409 | 业务状态不允许 |
| `100997` | 504 | 系统超时 |
| `100998` | 500 | 系统内部错误 |
| `130101` | 403 | 应用不存在或已停用 |
| `130102` | 401 | 密钥无效 |
| `130103` | 404 | 方法未开放 |
| `130104` | 429 | 请求超过限制（本期占位） |
| `100999` | 500 | 系统繁忙（基础设施类错误折叠后的对外码） |

下游错误优先读取 `google.rpc.ErrorInfo.reason`；取不到时按 gRPC Code 兜底。`100201`、`1003xx`、`1006xx`、`100998` 会折叠为 `100999 系统繁忙`，真实原因只写日志，不返回给第三方。

## 管理接口

与开放面同端口 `11005`，前缀 `/open-admin/v1`，使用 JWT 鉴权（`JwtAuth.AccessSecret`），并复用 `authctx` 中间件。

```http
POST /open-admin/v1/app/create        创建应用，secret 仅返回一次
GET  /open-admin/v1/app/list          列表（keyword/status/pageNum/pageSize）
POST /open-admin/v1/app/disable       启用或停用（status=1/2）
POST /open-admin/v1/app/secret/reset  重置密钥，旧密钥保留 24h
POST /open-admin/v1/app/sign-key/reveal  查看当前 signKey（受 SignKeyRevealEnabled 控制）
```

- 创建：`{"name":"示例应用","remark":"开放平台接入","signRequired":false}`，返回 `appId`、`secret`；`signRequired=true` 时还返回一次 `signKey`。
- 列表：`GET /open-admin/v1/app/list?keyword=&status=&pageNum=1&pageSize=20`。
- 停用：`{"appId":"app_xxx","status":2}`；可选 `signRequired` 同时切换是否要求签名。
- 重置：`{"appId":"app_xxx"}`；旧密钥 `status=2` 且 `expire_at=now+24h`，新密钥立即生效；应用开启签名时同时返回新的 `signKey`。
- 查看签名密钥：`{"appId":"app_xxx"}`，返回当前 `status=1` 记录的明文 `signKey`；只对 `signRequired=true` 的应用生效，需要配置 `SignKeyEncryptionKey` 且 `SignKeyRevealEnabled=true`，成功时记录 `app_id` 与操作人审计日志。
- `secret` 仍然只存 sha256 摘要，服务端无法找回，丢失只能重置；`signKey` 可通过上述管理接口查看（受开关控制），也可以通过本地 `cmd/keytool` 解密。
- 管理面同样使用 `HTTP 200 + {code,msg,data,timestamp}`：JWT 缺失/过期/签名错误返回 `200 {"code":100401,"msg":"未认证","timestamp":1761120000000}`，不再使用 HTTP 401。
- 管理面响应带 `Cache-Control: no-store`，避免凭据被浏览器或代理缓存。
- nginx 不对外转发 `/open-admin/`，该前缀应只在内网访问。

## gRPC 内网直连

Java 等内网调用方可跳过 HTTP 网关，直接调用 `app/openapi`：

- 契约：`app/openapi/openapi.proto`
- 服务：`openapi.OpenApiRpc/Invoke`
- `InvokeReq`：`app_id`、`method`、`data`（protojson string）、`tid`
- `InvokeRes`：`data`（成功时的 JSON string）；失败通过 gRPC status 返回，六位码位于 `ErrorInfo.reason`
- 直连路径不做鉴权，依赖网络隔离；`app_id` 由调用方填写，仅用于日志与后续限流扩展。
- 新增 method 不需要修改 proto，Java stub 只需生成一次。

## 启动与配置

```bash
# 转发服务（默认 Mock 模式，仅 dev/test）
cd app/openapi && go run . -f etc/openapi.yaml

# HTTP 网关
cd app/opengtw && go run . -f etc/opengtw.yaml
```

`app/openapi/etc/openapi.yaml`：

- `PodEngineConf`：podengine 地址与超时（默认 120s）。
- `BridgeModbusConf`：bridgemodbus 地址与超时（默认 120s）。
- `Mock: true`：直接返回固定数据，不连接下游；仅允许 `Mode: dev` / `test`，生产启动会失败。

`app/opengtw/etc/opengtw.yaml`：

- `DB.DataSource`：默认指向本地金仓 `zero` 库；生产建议独立 `resource` 库。仅在 dev/test 自动创建 `open_app`、`open_app_secret`。
- `OpenApiConf`：`app/openapi` 地址，超时 120s。
- `SignKeyEncryptionKey`：签名应用的签名 key 加密主密钥；仅存在 `signRequired=true` 应用时必填，使用配置文件直接配置，不提交真实密钥。
- `Redis`：签名 nonce 防重放存储；仅存在 `signRequired=true` 应用时必填。
- `JwtAuth.AccessSecret`：管理面 JWT 密钥，默认与 `gtw`/`livegtw` 一致，本地签发的 token 可直接复用。
- `MaxBytes: 1048576`：单请求体上限 1 MiB。

## 数据模型

| 表 | 说明 |
| --- | --- |
| `open_app` | 应用主表：`app_id`（varchar(40)）、`name`、`remark`、`status`、`sign_required`、`last_used_at` |
| `open_app_secret` | 密钥表：`app_id`（varchar(40)）、`secret_hash`（sha256 hex）、`sign_key_cipher`（加密后的签名密钥）、`status`、`expire_at` |

`status`：应用 `1-启用 2-停用`；密钥 `1-生效 2-待失效 3-已失效`。没有 SQL 迁移文件，表结构由 GORM 模型提供，开发/测试环境启动时自动迁移。

生产环境不会自动迁移，升级到本版本前需要手工执行（金仓 / PostgreSQL）：

```sql
ALTER TABLE open_app ALTER COLUMN app_id TYPE varchar(40);
ALTER TABLE open_app ADD COLUMN sign_required boolean NOT NULL DEFAULT false;
ALTER TABLE open_app_secret ALTER COLUMN app_id TYPE varchar(40);
ALTER TABLE open_app_secret ADD COLUMN sign_key_cipher text;
```

## 风险与边界

- 透传即契约：podengine / bridgemodbus proto 改名、改枚举会直接破坏第三方调用方。
- 管理面与开放面同端口，部署时必须确认 nginx 没有转发 `/open-admin/`。
- 单请求最长 120s，慢请求会长期占用连接。
- `Mock: true` 只允许 dev/test，生产配置评审必须检查该项。
- 本期开放 Podman 4 个只读方法与 Modbus 15 个 I/O 方法；Modbus 配置管理暂不对外。
- Modbus 写方法会改变现场设备状态，生产调用方必须做好地址和值校验。
