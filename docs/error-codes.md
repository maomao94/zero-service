# 平台错误码规范

本文只回答三个问题：

1. 错误码怎么编；
2. 平台错误码、gRPC Code、HTTP 状态三者怎么对应；
3. 服务端怎么返回、网关怎么输出、客户端怎么判断。

定义以 [`third_party/extproto.proto`](../third_party/extproto.proto) 为准，实现以 `common/errx`、`common/gtwx` 为准。

---

## 1. 错误分层与职责

错误从业务服务传到客户端，会经过下面几层，每层职责不同，不能混用：

| 层 | 载体 | 作用 | 取值来源 |
| --- | --- | --- | --- |
| **业务错误码** | gRPC `ErrorInfo.reason` / HTTP body `code` | 表达“业务上哪里错了”，是客户端判断失败的**唯一依据** | `third_party/extproto.proto` 中的 `Code` 枚举 |
| **gRPC Code** | gRPC status code | 表达“错误属于哪一类”，用于重试、监控和跨语言传输 | `google.golang.org/grpc/codes` 中的 `codes.Code` 常量，由 extproto 的 `http_code` 推导，始终是标准 gRPC Code `0-16` |
| **语义 HTTP** | extproto 的 `http_code` | 分类参考，决定 gRPC Code 和客户端处理方式 | `third_party/extproto.proto` |
| **网关实际 HTTP** | HTTP status line | 业务路由固定返回 `200`，不表达业务成败 | `common/gtwx` 统一写出 |
| **网关信封** | body `{code,msg,data,timestamp}` | 客户端唯一解析入口 | `common/gtwx` 统一写出 |

关键结论：

- **六位业务码只写入 `ErrorInfo.reason` / body `code`，永远不写入 gRPC Code。**
- **gRPC Code 在 Go 代码中必须使用 `google.golang.org/grpc/codes` 的常量**（`codes.OK`、`codes.InvalidArgument` 等），禁止直接写 `0`、`3`、`16` 等裸数字；本文表格里的 `(3)` 只用于阅读和对照。
- **网关业务路由统一返回 HTTP 200**，包括参数错误、鉴权失败和下游错误。客户端只认 `body.code == 0`，不要用 HTTP 状态判断成功与否。
- `http_code` 只用于推导 gRPC Code 和分类，不代表网关真实响应状态。

一次完整调用：

```text
业务服务
  extproto code = 100103
        │
        ▼
gRPC status
  Code   = codes.InvalidArgument (3)
  reason = "100103"
        │
        ▼
HTTP 网关
  HTTP 200
  body = {"code":100103,"msg":"参数不合法"}
        │
        ▼
客户端：code != 0，判定失败
```

---

## 2. 对应关系总表

按已知信息查表：

- 已知 **extproto 码**，查 [§5 全量错误码](#5-全量错误码)；
- 已知 **gRPC Code**，查 [§2.2 gRPC → extproto](#22-grpc--extproto兜底)；
- 已知 **语义 HTTP**，查 [§2.1 extproto → gRPC → HTTP](#21-extproto--grpc--http正向)。

### 2.1 extproto → gRPC → HTTP（正向）

这是**唯一权威映射**：每个 extproto 码的 `http_code` 决定它的 gRPC Code。

| 语义 HTTP (`http_code`) | gRPC Code（`codes.Code` 常量） | extproto 码（含义） | 客户端处理 |
| --- | --- | --- | --- |
| `200` | `codes.OK (0)` | `0 成功` | 成功 |
| `400` | `codes.InvalidArgument (3)` | `100101 参数错误`、`100102 缺少必要参数`、`100103 参数不合法`、`100501 业务处理失败` | 按 `msg` 修正请求或处理业务 |
| `401` | `codes.Unauthenticated (16)` | `100401 未认证`、`130102 密钥无效`、`130105 签名无效` | 重新获取或修正凭据 |
| `403` | `codes.PermissionDenied (7)` | `100402 无权限访问`、`130101 应用不存在或已停用` | 检查权限或应用状态，不要重试 |
| `404` | `codes.NotFound (5)` | `100202 记录不存在`、`100801 接口不存在`、`130103 方法未开放` | 检查资源 ID、路径或 method |
| `405` | `codes.Unimplemented (12)` | `100802 请求方法不允许` | 修改 HTTP 方法 |
| `408` | `codes.Canceled (1)` | `100996 请求已取消` | 按需重试 |
| `409` | `codes.Aborted (10)` | `100203 记录已存在`、`100204 数据冲突`、`100502 业务状态不允许`、`100503 重复操作` | 按 `msg` 处理，不要盲目重试 |
| `429` | `codes.ResourceExhausted (8)` | `100701 请求过于频繁`、`130104 请求超过限制` | 退避后重试 |
| `500` | `codes.Internal (13)` | `100201 数据库错误`、`100301 缓存错误`、`100302 缓存未命中`、`100998 系统内部错误`、`100999 未知错误` | 记录请求 ID，服务端排查 |
| `503` | `codes.Unavailable (14)` | `100303 消息队列错误`、`100601 远程调用失败`、`100602 第三方服务异常`、`100603 第三方接口错误` | 可重试；`100603` 按 `msg` 中的厂商信息处理 |
| `504` | `codes.DeadlineExceeded (4)` | `100997 系统超时` | 延长超时或排查下游 |

说明：

- `409` 对应多个业务码，正向传输时都会变成 `codes.Aborted`，**具体是哪一个以 `ErrorInfo.reason` / body `code` 为准**。
- `405 -> codes.Unimplemented`、`408 -> codes.Canceled`、`409 -> codes.Aborted` 是本项目的语义约定，不是 gRPC 官方 HTTP 映射。
- 反向分类时还有 `codes.OutOfRange -> 400`、`codes.AlreadyExists -> 409`、`codes.FailedPrecondition -> 409`、`codes.Unknown/codes.DataLoss -> 500`，这些 gRPC Code 不会由 extproto 的 `http_code` 正向产生，只用于兜底分类。

### 2.2 gRPC → extproto（兜底）

正常链路一定带 `ErrorInfo.reason`，网关优先使用它。只有在 `reason` 丢失或不是数字时，才按 gRPC Code 兜底：

| gRPC Code（`codes.Code` 常量） | 兜底 extproto 码 | 兜底含义 |
| --- | --- | --- |
| `codes.InvalidArgument (3)` / `codes.OutOfRange (11)` | `100101` | 参数错误 |
| `codes.NotFound (5)` | `100202` | 记录不存在 |
| `codes.AlreadyExists (6)` | `100203` | 记录已存在 |
| `codes.Aborted (10)` | `100204` | 数据冲突 |
| `codes.Unauthenticated (16)` | `100401` | 未认证 |
| `codes.PermissionDenied (7)` | `100402` | 无权限访问 |
| `codes.FailedPrecondition (9)` | `100502` | 业务状态不允许 |
| `codes.ResourceExhausted (8)` | `100701` | 请求过于频繁 |
| `codes.Unavailable (14)` | `100601` | 远程调用失败 |
| `codes.Unimplemented (12)` | `100802` | 请求方法不允许 |
| `codes.Canceled (1)` | `100996` | 请求已取消 |
| `codes.DeadlineExceeded (4)` | `100997` | 系统超时 |
| `codes.Internal (13)` / `codes.Unknown (2)` / `codes.DataLoss (15)` | `100998` | 系统内部错误 |
| 其他 | `100999` | 未知错误 / 系统繁忙 |

> 兜底会丢失同一类里的细分码。例如 `100203` 正向传输是 `codes.Aborted`，兜底只能还原成 `100204`。因此业务服务必须显式携带 `reason`，不能依赖网关猜测。

---

## 3. 错误码结构

平台错误码固定为 **ABCDEF** 六位数字：

```text
  1 30 101
  │ │  └── DEF：模块内错误（三位）
  │ └───── BC ：功能模块（两位）
  └─────── A  ：错误来源（一位）
```

### 3.1 A：错误来源

| A | 含义 | 说明 |
| --- | --- | --- |
| `0` | 成功 | 只允许 `000000` |
| `1` | 平台服务端 | 本仓库微服务与 HTTP/gRPC 网关 |
| `2-9` | 预留 | 当前不定义 |

平台码只使用 `A=0/1`，**不定义设备或第三方厂商码**。DJI 等厂商码是独立体系，见第 8 节。

### 3.2 BC：功能模块

只对 `A=1` 的码生效，按产品能力划分。当前启用：

| BC | 功能模块 | 覆盖范围 |
| --- | --- | --- |
| `00` | 通用 | 参数、数据、缓存、认证、外部依赖、网关、系统兜底 |
| `30` | 开放平台 | 应用、密钥、method、限流、签名 |

其他功能域按需申请，不提前预分配；`90-99` 预留。

### 3.3 DEF：模块内错误

| DEF | 用途 |
| --- | --- |
| `100-899` | 模块内具体错误，按申请顺序递增 |
| `900-999` | 系统兜底与对外折叠（取消、超时、内部、未知） |

通用模块 `00` 内部再分段，便于查找：

| DEF | 子域 |
| --- | --- |
| `100-199` | 参数 / 校验 |
| `200-299` | 数据 / 存储 |
| `300-399` | 缓存 / 消息 |
| `400-499` | 认证 / 授权 |
| `500-599` | 通用业务 |
| `600-699` | 外部依赖 |
| `700-799` | 限流 |
| `800-899` | 网关 / 传输 |
| `900-999` | 兜底 / 折叠 |

---

## 4. 命名与定义规范

Go / proto 枚举名格式：`_<A>_<BC>_<SEMANTIC>`，语义使用大写下划线。

```proto
_1_30_APP_SECRET_INVALID = 130102 [(name) = "密钥无效", (http_code) = 401];
```

规则：

1. 数字段必须与标识符一致，例如 `_1_30_...` 必须是 `130xxx`。
2. 新增码必须同时提供中文 `name` 和语义 `http_code`。
3. 修改文案可以；修改数字或 `http_code` 必须走评审。
4. `0` 是唯一成功码，禁止用 `0` 构造错误。
5. 对外 `msg` 必须可安全展示；SQL、地址、堆栈只写日志。

---

## 5. 全量错误码

下表是 `third_party/extproto.proto` 的完整镜像。`语义 HTTP` 只用于分类和推导 gRPC Code，网关业务路由实际仍返回 HTTP 200。

### 5.1 通用（BC=00）

| code | 名称 | 语义 HTTP | gRPC Code（`codes.Code` 常量） | 说明 |
| --- | --- | --- | --- | --- |
| `0` | 成功 | 200 | `codes.OK` | 唯一成功码 |
| `100101` | 参数错误 | 400 | `codes.InvalidArgument` | 请求参数无法解析 |
| `100102` | 缺少必要参数 | 400 | `codes.InvalidArgument` | 必填参数为空 |
| `100103` | 参数不合法 | 400 | `codes.InvalidArgument` | 格式、类型或范围不合法 |
| `100201` | 数据库错误 | 500 | `codes.Internal` | 数据库访问失败，可配置折叠 |
| `100202` | 记录不存在 | 404 | `codes.NotFound` | 指定资源不存在 |
| `100203` | 记录已存在 | 409 | `codes.Aborted` | 唯一约束冲突 |
| `100204` | 数据冲突 | 409 | `codes.Aborted` | 并发更新或状态冲突 |
| `100301` | 缓存错误 | 500 | `codes.Internal` | 缓存读写失败，可配置折叠 |
| `100302` | 缓存未命中 | 500 | `codes.Internal` | 缓存 key 不存在，可配置折叠 |
| `100303` | 消息队列错误 | 503 | `codes.Unavailable` | MQ 不可达或收发失败，可配置折叠 |
| `100401` | 未认证 | 401 | `codes.Unauthenticated` | JWT/凭据缺失、无效或过期 |
| `100402` | 无权限访问 | 403 | `codes.PermissionDenied` | 身份有效但权限不足 |
| `100501` | 业务处理失败 | 400 | `codes.InvalidArgument` | 业务逻辑拒绝处理 |
| `100502` | 业务状态不允许 | 409 | `codes.Aborted` | 当前状态不允许该操作 |
| `100503` | 重复操作 | 409 | `codes.Aborted` | 相同操作已执行 |
| `100601` | 远程调用失败 | 503 | `codes.Unavailable` | 下游 gRPC 不可达，可配置折叠 |
| `100602` | 第三方服务异常 | 503 | `codes.Unavailable` | 基础设施类第三方故障，可配置折叠 |
| `100603` | 第三方接口错误 | 503 | `codes.Unavailable` | 厂商返回错误码，**强制透传**，`msg` 使用固定模板 |
| `100701` | 请求过于频繁 | 429 | `codes.ResourceExhausted` | 平台限流 |
| `100801` | 接口不存在 | 404 | `codes.NotFound` | 网关路由不存在 |
| `100802` | 请求方法不允许 | 405 | `codes.Unimplemented` | 路径不支持当前 HTTP 方法 |
| `100996` | 请求已取消 | 408 | `codes.Canceled` | 调用方取消或连接中断 |
| `100997` | 系统超时 | 504 | `codes.DeadlineExceeded` | 下游超时 |
| `100998` | 系统内部错误 | 500 | `codes.Internal` | 服务内部异常，可配置折叠 |
| `100999` | 未知错误 | 500 | `codes.Internal` | 未登记码、未转换错误或折叠目标 |

### 5.2 开放平台（BC=30）

| code | 名称 | 语义 HTTP | gRPC Code（`codes.Code` 常量） | 说明 |
| --- | --- | --- | --- | --- |
| `130101` | 应用不存在或已停用 | 403 | `codes.PermissionDenied` | Basic 凭据中的应用不可用 |
| `130102` | 密钥无效 | 401 | `codes.Unauthenticated` | 缺少 Basic 凭据或 secret 不匹配 |
| `130103` | 方法未开放 | 404 | `codes.NotFound` | method 未在开放平台注册 |
| `130104` | 请求超过限制 | 429 | `codes.ResourceExhausted` | 开放平台限流 |
| `130105` | 签名无效 | 401 | `codes.Unauthenticated` | `X-Sign` 缺失或校验失败 |

> 当前没有专属码的功能域（如设备、会议、GIS、文件、AI、工业协议、容器、用户）按第 3.2 节归属另行申请，不跨域复用。

---

## 6. 服务端：构造与归一化

### 6.1 业务服务构造错误

```go
return nil, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID, "会议标题不能为空")
```

需要保留底层原因时：

```go
return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询会议失败")
```

- `NewErrorByPbCode` 把六位码写入 `ErrorInfo.reason`，并按 `http_code` 推导 gRPC Code。
- `NewErrorByPbCode(extproto.Code__0_OK)` 会直接返回内部错误，防止把成功码当错误。
- `NewErrorByPbCodeWrap` 遇到已带合法 extproto 码的 cause 时原样返回，禁止中间层重包覆盖。
- `cause` 只用于服务端日志和 `errors.Is/As`，不进入 gRPC message。

### 6.2 gRPC 出口归一化

`errx.ToGRPCStatus` 是 gRPC 服务端出口的最后一道归一化：

1. `errx.Error` 优先，保留业务码，不被 `context` 语义覆盖；
2. `xerrors.CodeMsg` 转为 `errx.Error`；
3. 其他实现了 `GRPCStatus()` 的错误保留原 status；
4. `context.Canceled` / `context.DeadlineExceeded` 分别保留为 `codes.Canceled` / `codes.DeadlineExceeded`；
5. 缺少 `ErrorInfo` 的 status 补一条兜底 detail；
6. 未转换的裸错误统一返回 `100999 系统繁忙`，原始信息只写日志。

### 6.3 判断错误码

```go
if errx.IsErrorByPbCode(err, extproto.Code__1_00_RECORD_NOT_EXIST) {
    // 记录不存在
}
```

支持 `errx.Error`、`xerrors.CodeMsg` 和带 `ErrorInfo.reason` 的 gRPC status，包装链上的错误也能识别。

---

## 7. 网关：返回契约

适用于 `gtw`、`livegtw`、`opengtw` 等 HTTP 网关的业务路由。代码落地与验收清单见 [`gateway-standard.md`](./gateway-standard.md)。

### 7.1 统一信封

```json
{"code": 0, "msg": "ok", "data": {}, "timestamp": 1760000000000}
```

| 字段 | 说明 |
| --- | --- |
| `code` | 六位平台码；`0` 成功，非 `0` 失败 |
| `msg` | 可展示文案；成功固定 `ok` |
| `data` | 成功时的业务数据；失败时省略 |
| `timestamp` | 网关生成响应时的 Unix 毫秒时间戳 |

失败示例：

```json
{"code": 100102, "msg": "缺少必要参数", "timestamp": 1760000000000}
```

### 7.2 HTTP 状态

**业务响应统一 HTTP 200**，包括鉴权失败、参数错误和下游错误。客户端必须以 `code == 0` 判断成功，不得以 HTTP 状态判断。

`X-Request-Id` 必须回显（缺失时由网关生成），排查问题时以该 ID 检索日志。

### 7.3 鉴权失败

| 场景 | 响应 |
| --- | --- |
| 缺少 / 错误 Basic 凭据 | `200 {"code":130102,"msg":"缺少或非法的 Basic 凭据"}` |
| JWT 缺失 / 过期 / 签名错误 | `200 {"code":100401,"msg":"未认证"}` |
| JWT 有效但缺少用户身份 | `200 {"code":100401,"msg":"缺少用户身份"}` |
| 应用不存在或已停用 | `200 {"code":130101,"msg":"应用不存在或已停用"}` |

### 7.4 网关自身错误

| 场景 | 响应 |
| --- | --- |
| 路由不存在 | `200 {"code":100801,"msg":"接口不存在"}` |
| 方法不允许 | `200 {"code":100802,"msg":"请求方法不允许"}` |
| 请求体超过上限 | `413`（go-zero 内置，空 body） |
| 请求超时 | `503` + 纯文本 `Request Timeout` |
| panic | `500`（go-zero 恢复中间件，空 body） |
| 反向代理不可达 | `502/504`（nginx 产生，非信封） |

业务路由内抛出的错误统一经 `errx.NormalizeError` 归一化：

- 错误码必须存在于 extproto 枚举，否则降级为 `100999` 并记录日志；
- 未转换的普通 `error` 一律返回 `100999 系统繁忙`，原文只写日志；
- 上表中 go-zero 内置中间件直接写出的框架级错误不经过该函数。

### 7.5 错误折叠

折叠是网关的可选展示策略，**默认不折叠，所有已登记码原样透传**。

- 在网关 `etc/*.yaml` 的 `Error.Fold` 中配置 `Enabled` 和 `Codes` 后，命中的码折叠为 `Target`（默认 `100999`）；
- `KeepMsg=false` 时文案统一为 `系统繁忙`；
- `100603` 由 `common/gtwx/fold.go` 强制透传，即使配置在 `Codes` 中也不会折叠；
- 折叠只影响 HTTP 信封，gRPC 传输永不折叠。

### 7.6 明确例外

以下接口不适用统一信封，按协议语义返回状态码：

- `POST /webhook/livekit`：LiveKit 依赖 HTTP 状态决定是否重推，验签失败返回 `401`，业务处理失败返回 `500`。
- 微信支付 / 退款回调（`gtw/internal/handler/pay`）：成功响应由 PowerWeChat 按微信协议写出，不套业务信封。
- gRPC 直连：使用 gRPC status + `ErrorInfo.reason`，不使用 HTTP 信封。

### 7.7 业务结果与平台错误的分工

- **纯命令接口**：没有业务扩展字段时，可以复用 `CommonRes`。
- **业务接口**：每个接口定义独立的 `XxxRes`，`code/message/tid/reason_code` 等结果字段保持扁平，业务字段直接追加在同一消息中。
- **不要**为了复用把业务接口的返回值替换成 `CommonRes`，也**不要**把 `CommonRes` 嵌套进业务响应。
- 平台错误走 gRPC error；设备拒绝、业务拒绝等“调用成功但结果是否定”的场景走独立 `XxxRes` 的扁平结果字段。

---

## 8. 第三方与设备错误码

### 8.1 基本原则

`extproto` 是唯一的平台错误码空间。其他系统的错误必须在**业务或适配层**完成转换，转换发生在调用厂商 SDK 的地方，不发生在网关。

| 厂商返回 | 处理 |
| --- | --- |
| 业务拒绝，但调用本身成功 | 转成业务字段（如 `CommonRes.reasonCode`），信封 `code` 保持 `0` |
| 第三方接口明确返回错误码，导致调用失败 | `100603 第三方接口错误`，厂商码与原文写入固定文案 |
| 平台故障（超时、连接失败、鉴权失败、限流、5xx） | 转成最接近的平台码（`100601`、`100602`、`100997`），以 error 返回 |
| 无法归类 | `100602 第三方服务异常`；连错误都拿不到时用 `100999` |

### 8.2 DJI 等厂商码

DJI 错误码（`third_party/dji_error_code.proto`）是**独立的厂商错误码体系**，与平台码不互通、不映射、不换号，也不作为平台码透传：

| 层 | 字段 | 规则 |
| --- | --- | --- |
| gRPC | `CommonRes.code` | DJI 命令结果：`0` 成功、`-1` 设备拒绝 |
| gRPC | `CommonRes.reason_code` | DJI 原始错误码，如 `312014`，原样返回 |
| HTTP 网关 | `data.reasonCode` | 原样返回；信封 `code` 仍为 `0` |

```json
{
  "code": 0,
  "msg": "ok",
  "data": {"code": -1, "reasonCode": 312014, "message": "设备升级中，请勿重复操作"}
}
```

规则：

- 设备命令的拒绝结果包装成 `CommonRes{Code: -1, ReasonCode: 312014}`，作为**正常响应**返回，不构造 error。
- DJI 等厂商码不写入 `ErrorInfo.reason`，也不进网关信封。
- 误写入网关的未登记码会降级为 `100999 系统繁忙`，原始值只写服务端日志。
- 需要跨服务传递设备结果时，继续使用 `CommonRes.reasonCode`（或响应体 `data.reasonCode`）。

### 8.3 第三方接口错误模板

第三方接口明确返回错误码时，统一使用 `errx.NewThirdPartyAPIError`，不要在各调用点手拼字符串：

```go
return errx.NewThirdPartyAPIError("dji", "312014", "设备升级中，请勿重复操作")
```

生成的 `msg` 使用跨语言 `%s` 模板，Go 与 Java 共用同一份模板：

```text
第三方接口错误，vendor=%s，code=%s，message=%s
```

样例：

```json
{"code":100603,"msg":"第三方接口错误，vendor=dji，code=312014，message=设备升级中，请勿重复操作"}
```

| 片段 | 含义 |
| --- | --- |
| `vendor=%s` | 厂商标识，原样保留，不做映射 |
| `code=%s` | 厂商原始错误码，原样保留 |
| `message=%s` | 厂商原始描述；为空时整段省略，使用无 message 模板 |

```java
String msg = String.format(
    "第三方接口错误，vendor=%s，code=%s，message=%s",
    vendor, vendorCode, vendorMessage);
```

```go
msg := fmt.Sprintf(
    "第三方接口错误，vendor=%s，code=%s，message=%s",
    vendor, vendorCode, vendorMessage)
```

业务侧可以用 `errx.ParseThirdPartyAPIError(msg)` 拿到三个值：

```go
vendor, code, message, ok := errx.ParseThirdPartyAPIError(resp.Msg)
```

其他语言按同一规则解析：以 `，` 分隔字段，再按 `=` 取值；只使用 `%s` 占位符，禁止 `%d`、`%v`、正则、`*`、`x`、`?`。

约束：

1. 只允许解析 `100603` 的固定模板；其他 `msg` 只给人看，客户端**不得解析**做分支。
2. 不放凭据、URL、堆栈、完整厂商响应体；构造器会压平空白，并在描述超过 200 字符时截断。
3. 厂商码不登记进 `extproto`，不做 1:1 映射；平台码只表达“失败来自第三方接口”。
4. 未在 extproto 登记的码一律降级 `100999`，原始值只进服务端日志。

---

## 9. 客户端处理建议

```java
// 伪代码
Response resp = http.post(...);
if (resp.code() != 0) {                 // 只认 code
    log.warn("requestId={} code={} msg={}", resp.requestId(), resp.code(), resp.msg());
    // 认证类：100401 / 130101 / 130102 / 130105
    // 可重试：100601 / 100602 / 100701 / 100997 / 100999
}
```

- 不要用 `HTTP 401` 触发重新登录，改用 `code == 100401 || code == 130101 || code == 130102 || code == 130105`。
- 日志必须带 `X-Request-Id`。
- 未知 `code` 按失败处理，保留原始 `msg`。
- 只有 `100603` 的 `msg` 可以按固定模板解析，其余 `msg` 不得用于业务分支。

---

## 10. 常见问题

**Q1：gRPC 返回 `codes.InvalidArgument`，为什么 HTTP 还是 200？**

两者不在同一层。gRPC Code 是传输分类，网关拿到后统一写 `HTTP 200 + {code,msg,data,timestamp}`；业务失败只体现在 `body.code`。

**Q2：`http_code` 和网关实际 HTTP 状态是一回事吗？**

不是。`http_code` 是 extproto 上的语义标记，用来推导 gRPC Code 和分类；网关业务路由固定返回 200。只有 webhook 等明确例外才按协议返回 4xx/5xx。

**Q3：`100203` 和 `100204` 都是 409，怎么区分？**

`http_code` 只到 409 这一级，细分靠 `ErrorInfo.reason` / `body.code`。如果 `reason` 丢失，网关只能按 gRPC Code 兜底成 `100204`。

**Q4：gRPC 直连也返回 HTTP 200 吗？**

不返回。直连不走 HTTP 网关，看到的是 gRPC status + `ErrorInfo.reason`，没有 HTTP 信封。

---

## 11. 演进规则

1. **数字冻结**：码值一旦对外发布即冻结，即使文案调整，数字也不变。
2. **只追加**：新错误从对应功能域的空闲 DEF 递增；功能域用尽后申请新 BC 段。
3. **不复用**：删除的错误码保留占位，不分配给新语义。
4. **不跨域造码**：同一语义复用同一码；服务差异体现在 `msg` 和日志。
5. **标识符纠正**：只允许“数字不变、名称修正”。
6. **废弃流程**：文档标注 `deprecated`，至少保留两个发布周期。

---

## 12. 落地状态与关键文件

| 项 | 状态 |
| --- | --- |
| `extproto.proto` 码表与 `http_code` | 已定义 |
| `opengtw` 统一信封与 `code=0` | 已具备；管理面 JWT 失败已改为信封 |
| `livegtw` 统一信封与 `code=0` | 已对齐：鉴权、参数、下游错误均返回六位码 |
| `livegtw` webhook | 保持状态码语义（协议例外） |
| `gtw`（BFF） | 成功信封、错误码归一化与 `Error.Fold` 配置均已具备 |
| 其余网关（`aigtw`/`ssegtw`/`oryxgtw`/`bridgegtw`） | 待按第 7 节接入 |

| 文件 | 作用 |
| --- | --- |
| `third_party/extproto.proto` | 平台错误码定义与 `http_code` |
| `common/errx/error.go` | `NewErrorByPbCode` / `NewErrorByPbCodeWrap` / `IsErrorByPbCode` |
| `common/errx/codec.go` | `CodeFromStatus`、`IsDefinedCode`、`ErrorMessage` |
| `common/errx/httpstatus.go` | `http_code` ↔ gRPC Code 映射 |
| `common/errx/normalize.go` | `PublicError`、`NormalizeError` |
| `common/errx/status.go` | `ToGRPCStatus` gRPC 出口归一化 |
| `common/errx/thirdparty.go` | 第三方接口错误模板 |
| `common/gtwx/response.go` | HTTP 统一信封写出口 |
| `common/gtwx/fold.go` | 网关折叠配置，默认透传 |
| `docs/gateway-standard.md` | 网关落地规范与验收清单 |
