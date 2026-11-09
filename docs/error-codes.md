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
  extproto code = 100009
        │
        ▼
gRPC status
  Code   = codes.InvalidArgument (3)
  reason = "100009"
        │
        ▼
HTTP 网关
  HTTP 200
  body = {"code":100009,"msg":"参数不合法"}
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

| 语义 HTTP (`http_code`) | gRPC Code（`codes.Code` 常量） | 典型场景 | 客户端处理 |
| --- | --- | --- | --- |
| `200` | `codes.OK (0)` | `0 成功` | 成功 |
| `400` | `codes.InvalidArgument (3)` | 参数错误、参数校验、业务校验 | 按 `msg` 修正请求 |
| `401` | `codes.Unauthenticated (16)` | 未认证、凭据过期/无效、签名/时间戳错误 | 重新获取或修正凭据 |
| `403` | `codes.PermissionDenied (7)` | 无权限、账号禁用、租户不匹配、IP 限制 | 检查权限或应用状态，不要重试 |
| `404` | `codes.NotFound (5)` | 记录、路由、文件、配置不存在 | 检查资源 ID、路径或 method |
| `405` | `codes.Unimplemented (12)` | 请求方法不允许 | 修改 HTTP 方法 |
| `408` | `codes.Canceled (1)` | 请求取消、请求超时 | 按需重试 |
| `409` | `codes.Aborted (10)` | 数据冲突、版本冲突、业务状态/规则冲突 | 按 `msg` 处理，不要盲目重试 |
| `413` | `codes.ResourceExhausted (8)` | 请求体过大、文件过大 | 缩小请求后重试 |
| `415` | `codes.InvalidArgument (3)` | 媒体类型或文件类型不支持 | 修改 Content-Type / 文件类型 |
| `429` | `codes.ResourceExhausted (8)` | 平台、业务、外部依赖限流 | 退避后重试 |
| `500` | `codes.Internal (13)` | 数据库、缓存、序列化、内部异常 | 记录请求 ID，服务端排查 |
| `501` | `codes.Unimplemented (12)` | 功能未实现 | 换用已支持的能力或升级版本 |
| `502` | `codes.Unavailable (14)` | 网关错误、第三方接口错误、Webhook 失败 | 按 `msg` 处理；`100058` 看厂商信息 |
| `503` | `codes.Unavailable (14)` | 下游、第三方、MQ、DNS 不可用 | 可重试 |
| `504` | `codes.DeadlineExceeded (4)` | 系统、下游、网关超时 | 延长超时或排查下游 |

说明：

- 多个 HTTP 状态会收敛到同一个 gRPC Code：`400/415 -> InvalidArgument`、`405/501 -> Unimplemented`、`413/429 -> ResourceExhausted`、`502/503 -> Unavailable`。**具体错误以 `ErrorInfo.reason` / body `code` 为准。**
- `405 -> codes.Unimplemented`、`408 -> codes.Canceled`、`409 -> codes.Aborted` 是本项目的语义约定，不是 gRPC 官方 HTTP 映射。
- 反向分类时还有 `codes.OutOfRange -> 400`、`codes.AlreadyExists -> 409`、`codes.FailedPrecondition -> 409`、`codes.Unknown/codes.DataLoss -> 500`，这些 gRPC Code 不会由 extproto 的 `http_code` 正向产生，只用于兜底分类。
- 全量码表见 [§5](#5-全量错误码)。

### 2.2 gRPC → extproto（兜底）

正常链路一定带 `ErrorInfo.reason`，网关优先使用它。只有在 `reason` 丢失或不是数字时，才按 gRPC Code 兜底：

| gRPC Code（`codes.Code` 常量） | 兜底 extproto 码 | 兜底含义 |
| --- | --- | --- |
| `codes.InvalidArgument (3)` / `codes.OutOfRange (11)` | `100007` | 参数错误 |
| `codes.NotFound (5)` | `100021` | 记录不存在 |
| `codes.AlreadyExists (6)` | `100022` | 记录已存在 |
| `codes.Aborted (10)` | `100023` | 数据冲突 |
| `codes.Unauthenticated (16)` | `100036` | 未认证 |
| `codes.PermissionDenied (7)` | `100039` | 无权限访问 |
| `codes.FailedPrecondition (9)` | `100045` | 业务状态不允许 |
| `codes.ResourceExhausted (8)` | `100064` | 请求过于频繁 |
| `codes.Unavailable (14)` | `100054` | 远程调用失败 |
| `codes.Unimplemented (12)` | `100068` | 请求方法不允许 |
| `codes.Canceled (1)` | `100004` | 请求已取消 |
| `codes.DeadlineExceeded (4)` | `100003` | 系统超时 |
| `codes.Internal (13)` / `codes.Unknown (2)` / `codes.DataLoss (15)` | `100001` | 系统内部错误 |
| 其他 | `100002` | 未知错误 / 系统繁忙 |

> 兜底会丢失同一类里的细分码。例如 `100022` 正向传输是 `codes.Aborted`，兜底只能还原成 `100023`。因此业务服务必须显式携带 `reason`，不能依赖网关猜测。

---

## 3. 错误码结构

平台错误码固定为 **ABCDEF** 六位数字：

```text
  1 30 101
  │ │  └── DEF：模块内唯一编号（三位）
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

### 3.3 DEF：模块内唯一编号

| DEF | 用途 |
| --- | --- |
| `001-999` | 模块内唯一编号，按需分配，不要求连续 |
| `000` | 保留 |

DEF 不承载业务语义：不要按参数、数据、缓存等类别去固定百位或十位，也不要求连续。当前码表已按模块内 `001` 起连续分配；未使用的号就是后续可用容量。

#### 3.3.1 新增码的分配规则

1. **DEF 是模块内唯一编号**：新码取任意空闲号即可，建议按申请顺序追加；可以按业务在注释里分组，但数字本身不带业务含义。
2. **单模块 999 个码**：`BC=00-89` 共 90 个模块，理论容量约 9 万个码；单模块用尽后申请新 `BC`，不重用废弃号。
3. **上线后冻结**：本次为上线前重排，码表已归一为连续编号；上线后只追加，不再重排或复用。
4. **不编码 HTTP/gRPC**：HTTP 状态放 `http_code`，gRPC Code 由映射推导，禁止把 4xx/5xx 或 0-16 编进 DEF。

> 本次已在上线前统一重排；上线后按“数字冻结、只追加”执行，不再重排或复用。

---

## 4. 命名与定义规范

Go / proto 枚举名格式：`_<A>_<BC>_<SEMANTIC>`，语义使用大写下划线。

```proto
_1_30_APP_SECRET_INVALID = 130002 [(name) = "应用密钥无效", (http_code) = 401];
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
| `100001` | 系统内部错误 | 500 | `codes.Internal` | 可配置折叠 |
| `100002` | 未知错误 | 500 | `codes.Internal` | 折叠目标 / 未登记码兜底 |
| `100003` | 系统超时 | 504 | `codes.DeadlineExceeded` | — |
| `100004` | 请求已取消 | 408 | `codes.Canceled` | — |
| `100005` | 功能未实现 | 501 | `codes.Unimplemented` | — |
| `100006` | 服务不可用 | 503 | `codes.Unavailable` | — |
| `100007` | 参数错误 | 400 | `codes.InvalidArgument` | — |
| `100008` | 缺少必要参数 | 400 | `codes.InvalidArgument` | — |
| `100009` | 参数不合法 | 400 | `codes.InvalidArgument` | — |
| `100010` | 参数类型错误 | 400 | `codes.InvalidArgument` | — |
| `100011` | 参数超出范围 | 400 | `codes.InvalidArgument` | — |
| `100012` | 参数格式错误 | 400 | `codes.InvalidArgument` | — |
| `100013` | 请求格式错误 | 400 | `codes.InvalidArgument` | — |
| `100014` | 请求体过大 | 413 | `codes.ResourceExhausted` | — |
| `100015` | 媒体类型不支持 | 415 | `codes.InvalidArgument` | — |
| `100016` | 分页参数错误 | 400 | `codes.InvalidArgument` | — |
| `100017` | 幂等键无效 | 400 | `codes.InvalidArgument` | — |
| `100018` | 请求头不合法 | 400 | `codes.InvalidArgument` | — |
| `100019` | 数据库错误 | 500 | `codes.Internal` | 可配置折叠 |
| `100020` | 数据约束冲突 | 409 | `codes.Aborted` | — |
| `100021` | 记录不存在 | 404 | `codes.NotFound` | — |
| `100022` | 记录已存在 | 409 | `codes.Aborted` | — |
| `100023` | 数据冲突 | 409 | `codes.Aborted` | — |
| `100024` | 数据损坏 | 500 | `codes.Internal` | — |
| `100025` | 事务失败 | 500 | `codes.Internal` | — |
| `100026` | 序列化失败 | 500 | `codes.Internal` | — |
| `100027` | 反序列化失败 | 500 | `codes.Internal` | — |
| `100028` | 版本冲突 | 409 | `codes.Aborted` | — |
| `100029` | 乐观锁冲突 | 409 | `codes.Aborted` | — |
| `100030` | 缓存错误 | 500 | `codes.Internal` | 可配置折叠 |
| `100031` | 消息队列错误 | 503 | `codes.Unavailable` | 可配置折叠 |
| `100032` | 消息发送失败 | 503 | `codes.Unavailable` | — |
| `100033` | 消息消费失败 | 500 | `codes.Internal` | — |
| `100034` | 消息处理超时 | 504 | `codes.DeadlineExceeded` | — |
| `100035` | 事件处理失败 | 500 | `codes.Internal` | — |
| `100036` | 未认证 | 401 | `codes.Unauthenticated` | — |
| `100037` | 凭据已过期 | 401 | `codes.Unauthenticated` | — |
| `100038` | 凭据无效 | 401 | `codes.Unauthenticated` | — |
| `100039` | 无权限访问 | 403 | `codes.PermissionDenied` | — |
| `100040` | 账号已禁用 | 403 | `codes.PermissionDenied` | — |
| `100041` | 租户不匹配 | 403 | `codes.PermissionDenied` | — |
| `100042` | 权限范围不足 | 403 | `codes.PermissionDenied` | — |
| `100043` | 来源 IP 不在白名单 | 403 | `codes.PermissionDenied` | — |
| `100044` | 业务处理失败 | 409 | `codes.Aborted` | — |
| `100045` | 业务状态不允许 | 409 | `codes.Aborted` | — |
| `100046` | 重复操作 | 409 | `codes.Aborted` | — |
| `100047` | 业务校验失败 | 400 | `codes.InvalidArgument` | — |
| `100048` | 业务规则不满足 | 409 | `codes.Aborted` | — |
| `100049` | 业务配额不足 | 429 | `codes.ResourceExhausted` | — |
| `100050` | 业务已过期 | 409 | `codes.Aborted` | — |
| `100051` | 业务已锁定 | 409 | `codes.Aborted` | — |
| `100052` | 操作不允许 | 403 | `codes.PermissionDenied` | — |
| `100053` | 资源使用中 | 409 | `codes.Aborted` | — |
| `100054` | 远程调用失败 | 503 | `codes.Unavailable` | 可配置折叠 |
| `100055` | 远程调用超时 | 504 | `codes.DeadlineExceeded` | — |
| `100056` | 下游服务不可用 | 503 | `codes.Unavailable` | — |
| `100057` | 第三方服务异常 | 503 | `codes.Unavailable` | 可配置折叠 |
| `100058` | 第三方接口错误 | 502 | `codes.Unavailable` | 厂商返回错误码，**强制透传**，`msg` 使用固定模板 |
| `100059` | 第三方超时 | 504 | `codes.DeadlineExceeded` | — |
| `100060` | 第三方鉴权失败 | 502 | `codes.Unavailable` | — |
| `100061` | 外部限流 | 429 | `codes.ResourceExhausted` | — |
| `100062` | Webhook 投递失败 | 502 | `codes.Unavailable` | — |
| `100063` | DNS 解析失败 | 503 | `codes.Unavailable` | — |
| `100064` | 请求过于频繁 | 429 | `codes.ResourceExhausted` | — |
| `100065` | 配额已用尽 | 429 | `codes.ResourceExhausted` | — |
| `100066` | 并发超限 | 429 | `codes.ResourceExhausted` | — |
| `100067` | 接口不存在 | 404 | `codes.NotFound` | — |
| `100068` | 请求方法不允许 | 405 | `codes.Unimplemented` | — |
| `100069` | 网关错误 | 502 | `codes.Unavailable` | — |
| `100070` | 网关超时 | 504 | `codes.DeadlineExceeded` | — |
| `100071` | 协议错误 | 400 | `codes.InvalidArgument` | — |
| `100072` | TLS 错误 | 400 | `codes.InvalidArgument` | — |
| `100073` | 请求超时 | 408 | `codes.Canceled` | — |
| `100074` | 上游不可用 | 503 | `codes.Unavailable` | — |
| `100075` | 文件不存在 | 404 | `codes.NotFound` | — |
| `100076` | 文件过大 | 413 | `codes.ResourceExhausted` | — |
| `100077` | 文件类型不允许 | 415 | `codes.InvalidArgument` | — |
| `100078` | 文件上传失败 | 500 | `codes.Internal` | — |
| `100079` | 文件下载失败 | 503 | `codes.Unavailable` | — |
| `100080` | 文件损坏 | 500 | `codes.Internal` | — |
| `100081` | 文件解析失败 | 400 | `codes.InvalidArgument` | — |
| `100082` | 配置不存在 | 404 | `codes.NotFound` | — |
| `100083` | 配置不合法 | 500 | `codes.Internal` | — |
| `100084` | 配置解析失败 | 500 | `codes.Internal` | — |
| `100085` | 配置重载失败 | 500 | `codes.Internal` | — |

### 5.2 开放平台（BC=30）

| code | 名称 | 语义 HTTP | gRPC Code（`codes.Code` 常量） | 说明 |
| --- | --- | --- | --- | --- |
| `130001` | 应用不存在 | 403 | `codes.PermissionDenied` | — |
| `130002` | 应用密钥无效 | 401 | `codes.Unauthenticated` | — |
| `130003` | 应用已禁用 | 403 | `codes.PermissionDenied` | — |
| `130004` | 方法未开放 | 404 | `codes.NotFound` | — |
| `130005` | 请求超过限制 | 429 | `codes.ResourceExhausted` | — |
| `130006` | 签名无效 | 401 | `codes.Unauthenticated` | — |
| `130007` | 缺少签名 | 401 | `codes.Unauthenticated` | — |
| `130008` | 时间戳无效 | 401 | `codes.Unauthenticated` | — |
| `130009` | 随机数无效 | 401 | `codes.Unauthenticated` | — |
| `130010` | 应用 IP 不在白名单 | 403 | `codes.PermissionDenied` | — |
| `130011` | 应用配额不足 | 429 | `codes.ResourceExhausted` | — |
| `130012` | 应用无权限 | 403 | `codes.PermissionDenied` | — |
| `130013` | 接口版本不支持 | 400 | `codes.InvalidArgument` | — |
| `130014` | 请求已过期 | 401 | `codes.Unauthenticated` | — |

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
6. 未转换的裸错误统一返回 `100002 系统繁忙`，原始信息只写日志。

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
{"code": 100008, "msg": "缺少必要参数", "timestamp": 1760000000000}
```

### 7.2 HTTP 状态

**业务响应统一 HTTP 200**，包括鉴权失败、参数错误和下游错误。客户端必须以 `code == 0` 判断成功，不得以 HTTP 状态判断。

`X-Request-Id` 必须回显（缺失时由网关生成），排查问题时以该 ID 检索日志。

### 7.3 鉴权失败

| 场景 | 响应 |
| --- | --- |
| 缺少 / 错误 Basic 凭据 | `200 {"code":130002,"msg":"缺少或非法的 Basic 凭据"}` |
| JWT 缺失 / 过期 / 签名错误 | `200 {"code":100036,"msg":"未认证"}` |
| JWT 有效但缺少用户身份 | `200 {"code":100036,"msg":"缺少用户身份"}` |
| 应用不存在或已停用 | `200 {"code":130001,"msg":"应用不存在或已停用"}` |

### 7.4 网关自身错误

| 场景 | 响应 |
| --- | --- |
| 路由不存在 | `200 {"code":100067,"msg":"接口不存在"}` |
| 方法不允许 | `200 {"code":100068,"msg":"请求方法不允许"}` |
| 请求体超过上限 | `413`（go-zero 内置，空 body） |
| 请求超时 | `503` + 纯文本 `Request Timeout` |
| panic | `500`（go-zero 恢复中间件，空 body） |
| 反向代理不可达 | `502/504`（nginx 产生，非信封） |

业务路由内抛出的错误统一经 `errx.NormalizeError` 归一化：

- 错误码必须存在于 extproto 枚举，否则降级为 `100002` 并记录日志；
- 未转换的普通 `error` 一律返回 `100002 系统繁忙`，原文只写日志；
- 上表中 go-zero 内置中间件直接写出的框架级错误不经过该函数。

### 7.5 错误折叠

折叠是网关的可选展示策略，**默认不折叠，所有已登记码原样透传**。

- 在网关 `etc/*.yaml` 的 `Error.Fold` 中配置 `Enabled` 和 `Codes` 后，命中的码折叠为 `Target`（默认 `100002`）；
- `KeepMsg=false` 时文案统一为 `系统繁忙`；
- `100058` 由 `common/gtwx/fold.go` 强制透传，即使配置在 `Codes` 中也不会折叠；
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
| 第三方接口明确返回错误码，导致调用失败 | `100058 第三方接口错误`，厂商码与原文写入固定文案 |
| 平台故障（超时、连接失败、鉴权失败、限流、5xx） | 转成最接近的平台码（`100054`、`100057`、`100003`），以 error 返回 |
| 无法归类 | `100057 第三方服务异常`；连错误都拿不到时用 `100002` |

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
- 误写入网关的未登记码会降级为 `100002 系统繁忙`，原始值只写服务端日志。
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
{"code":100058,"msg":"第三方接口错误，vendor=dji，code=312014，message=设备升级中，请勿重复操作"}
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

1. 只允许解析 `100058` 的固定模板；其他 `msg` 只给人看，客户端**不得解析**做分支。
2. 不放凭据、URL、堆栈、完整厂商响应体；构造器会压平空白，并在描述超过 200 字符时截断。
3. 厂商码不登记进 `extproto`，不做 1:1 映射；平台码只表达“失败来自第三方接口”。
4. 未在 extproto 登记的码一律降级 `100002`，原始值只进服务端日志。

---

## 9. 客户端处理建议

```java
// 伪代码
Response resp = http.post(...);
if (resp.code() != 0) {                 // 只认 code
    log.warn("requestId={} code={} msg={}", resp.requestId(), resp.code(), resp.msg());
    // 认证类：100036 / 130001 / 130002 / 130006
    // 可重试：100054 / 100057 / 100064 / 100003 / 100002
}
```

- 不要用 `HTTP 401` 触发重新登录，改用 `code == 100036 || code == 130001 || code == 130002 || code == 130006`。
- 日志必须带 `X-Request-Id`。
- 未知 `code` 按失败处理，保留原始 `msg`。
- 只有 `100058` 的 `msg` 可以按固定模板解析，其余 `msg` 不得用于业务分支。

---

## 10. 常见问题

**Q1：gRPC 返回 `codes.InvalidArgument`，为什么 HTTP 还是 200？**

两者不在同一层。gRPC Code 是传输分类，网关拿到后统一写 `HTTP 200 + {code,msg,data,timestamp}`；业务失败只体现在 `body.code`。

**Q2：`http_code` 和网关实际 HTTP 状态是一回事吗？**

不是。`http_code` 是 extproto 上的语义标记，用来推导 gRPC Code 和分类；网关业务路由固定返回 200。只有 webhook 等明确例外才按协议返回 4xx/5xx。

**Q3：`100022` 和 `100023` 都是 409，怎么区分？**

`http_code` 只到 409 这一级，细分靠 `ErrorInfo.reason` / `body.code`。如果 `reason` 丢失，网关只能按 gRPC Code 兜底成 `100023`。

**Q4：gRPC 直连也返回 HTTP 200 吗？**

不返回。直连不走 HTTP 网关，看到的是 gRPC status + `ErrorInfo.reason`，没有 HTTP 信封。

---

## 11. 演进规则

1. **数字冻结**：上线前允许重排；上线后码值一旦对外发布即冻结，文案调整不改数字。
2. **只追加**：新错误取模块内任意空闲 DEF；单个模块用尽 999 个号后申请新 BC，不复用废弃号。
3. **不复用**：删除的错误码保留占位，不分配给新语义。
4. **不跨域造码**：同一语义复用同一码；服务差异体现在 `msg` 和日志。
5. **标识符纠正**：只允许“数字不变、名称修正”。
6. **废弃流程**：文档标注 `deprecated`，至少保留两个发布周期。
7. **规则守护**：新增码必须通过 `common/errx/codetable_test.go` 的结构校验，包括 A/BC/DEF 一致性、模块白名单、`http_code` 白名单和成功码唯一性。

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
| `common/errx/codetable_test.go` | 码表结构规则守护测试（A/BC/DEF、模块、`http_code`、成功码唯一） |
| `common/gtwx/response.go` | HTTP 统一信封写出口 |
| `common/gtwx/fold.go` | 网关折叠配置，默认透传 |
| `docs/gateway-standard.md` | 网关落地规范与验收清单 |
