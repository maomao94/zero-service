# 编码规范

> 全局协作、安全、命名和 Git 边界。go-zero 代码生成、服务结构和公共组件清单以 [`go-zero-conventions.md`](./go-zero-conventions.md) 为准。

## 技术栈摘要

- Go 1.25+，go-zero，gRPC，grpc-gateway，Protocol Buffers，go-zero `.api`。
- AI 相关服务使用 CloudWeGo Eino / Eino ADK / MCP / OpenAI-compatible API。
- 契约变更先改 `.api` / `.proto`，再执行服务目录的 `gen.sh`，详细流程见 [`go-zero-conventions.md`](./go-zero-conventions.md)。

## AI 协作纪律

- 默认使用中文沟通、提问、任务清单和总结；用户明确要求其他语言时再切换。
- 修改前先阅读相邻实现、配置、Trellis task 和相关 spec，不基于猜测改动。
- 能通过搜索代码和读取配置推断的低风险事项自主推进；意图多解且影响大、不可逆操作、缺关键参数或规则冲突时再提问。
- 按最小影响范围修改，避免无关重构、无关格式化和大范围重排。
- 不主动创建额外文档或注释；但 `.api`、`.proto`、导出公共能力和复杂协议字段必须保留必要说明。
- 总结时说明改动内容、影响范围、已执行验证和未验证原因。

## 平台和配置边界

- `.opencode/agents/**`、`.opencode/skills/**`、`.opencode/commands/**`、`.opencode/plugins/**`、`.opencode/lib/**` 属于 Trellis/OpenCode 适配层，除非用户明确要求修改适配器，否则不要改。
- `.opencode/rules/**` 与 `.aiassistant/rules/**` 分别供 OpenCode 和 GoLand AI 读取；修改规则时两边保持同步。
- 不把个人绝对路径、API Key、Token、本机账号或私有基础设施写进 spec、规则、提交信息或总结。
- Markdown LSP 未配置时，文档任务至少执行 `git diff --check` 并说明原因；不要在本规范保留安装教程。需要配置 LSP 时按用户要求单独处理。

## 命名摘要

| 场景 | 命名 |
| --- | --- |
| API 网关请求 | `xxxRequest` |
| API 网关响应 | `xxxResponse` |
| gRPC 请求 | `xxxReq` |
| gRPC 响应 | `xxxRes` |

- 请求和响应必须成对出现，并保持注释完整。
- Go 包名、文件名、结构体和函数跟随相邻实现和 Go/go-zero 习惯。
- 禁止 Java 风格命名、异常处理、无意义 getter/setter 和过度封装。

## 编码边界

- Handler/Server 负责参数接收、校验、调用 Logic 和返回结果；业务编排放在 Logic。
- 跨服务复用能力沉淀到 `common/`，服务内部有状态逻辑保留在对应服务 `internal/`。
- 新增依赖前先检查 `go.mod`、相邻模块和现有封装，不重复引入功能相近的库。
- 涉及数据库、Redis、消息队列、MQTT、OSS、Docker、Eino 或 DJI Cloud API 时，优先复用已有 model、client、cache、config、SDK 和 `common/` 封装。
- 工具函数、复杂协议转换和关键业务分支应有单元测试；生成代码非必要不写自定义测试。

## Go 泛型约定

Go 1.18+ 泛型用于消除重复的类型转换和切片操作。约束类型定义在使用该约束的包中。

### 约束定义

```go
// 数值类型约束（用于字节/寄存器转换）
type Integer interface {
    ~int16 | ~uint16 | ~int32 | ~uint32
}
```

### 泛型切片转换

```go
// ConvertSlice 泛型切片转换，消除 XxxSliceToYyySlice 重复模式
func ConvertSlice[From Integer, To Integer](values []From, convert func(From) To) []To {
    result := make([]To, len(values))
    for i, v := range values {
        result[i] = convert(v)
    }
    return result
}
```

Usage:

```go
// 替代 Uint16SliceToUint32Slice、Int16SliceToInt32Slice 等重复函数
uint32s := bytex.ConvertSlice(uint16s, func(v uint16) uint32 { return uint32(v) })
int16s := bytex.ConvertSlice(int32s, func(v int32) int16 { return int16(v) })
```

### 泛型使用原则

- 泛型用于**消除重复模式**，不是为了炫技。
- 约束类型放在使用它的包中（如 `bytex.Integer`），不建跨包共享约束文件。
- 转换函数保持简单：一个 `convert func(From) To` 参数，不加更多泛型层。
- 泛型函数的调用方负责类型安全（如 `int16(v)` 截断是预期行为）。

## common/ 包复用原则

- 工具函数只在一个 `common/` 包中定义，不在其他包中复制。
- `common/bytex/` 是字节/寄存器转换的唯一来源（`tool/util.go` 中的重复已清除）。
- 新增 `common/` 包前先搜索是否已有类似功能。
- `common/tool/` 是混合工具包，不适合放特定领域的工具函数。

### Convention: Client Option 构造配置边界

**What**: 公共 client / SDK 的函数式 option 必须写入 `XxxOptions` 构造配置结构体，而不是直接写入运行态 `XxxClient` 或未导出实现结构体。

**Why**: option 属于构造参数，直接接收 `*Client` 会把配置解析和运行态对象耦合在一起；后续 client 增加连接池、锁、缓存或状态字段时，option 容易绕过构造边界并误改运行态状态。

**Contract**:

```go
type ClientOptions struct {
    Engine Engine
}

type ClientOption func(*ClientOptions)

func NewClient(opts ...ClientOption) *Client {
    o := &ClientOptions{}
    for _, opt := range opts {
        opt(o)
    }
    return &Client{engine: o.Engine}
}
```

**Good/Base/Bad Cases**:

- Good: `WithEngine(e Engine) ClientOption` 只设置 `ClientOptions.Engine`，`NewClient` 负责把配置映射到 `Client`。
- Base: 私有下载、请求、传输选项可以使用小写内部配置结构体，例如 `type downloadOptions struct`。
- Bad: `type ClientOption func(*Client)`，让 option 直接修改运行态 client 内部字段。

**Tests Required**: 修改 option 模式时，至少运行目标包测试并断言默认值、自定义 option、nil/默认 engine 路径行为不变。

**Wrong vs Correct**:

Wrong:

```go
type ClientOption func(*Client)
```

Correct:

```go
type ClientOption func(*ClientOptions)
```

## 安全规则

- 不新增、打印或记录明文密码、Token、密钥、认证头、证书、数据库连接串、对象存储配置、手机号、身份证号、内网地址或个人本地路径。
- 写入规则、文档、提交信息或总结时，对敏感信息脱敏；必要时改写为“项目内目录”“本地仓库配置”“用户级配置”等通用描述。
- 不主动执行部署、发布、远程上传、删除数据、修改共享基础设施等有外部副作用的操作，除非用户明确要求并给出目标环境。

## Git 规范

- 不主动执行 `git commit`；只有用户明确要求提交时才提交。
- 提交前必须查看实际 diff，提交说明结合改动内容，不写泛泛模板。
- 提交标题优先中文，可使用 conventional commit 前缀，例如 `feat: 新增火情任务状态同步`。
- 不在提交信息中写敏感信息、内网地址、账号、Token、密钥或完整异常堆栈。
