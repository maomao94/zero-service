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

## 安全规则

- 不新增、打印或记录明文密码、Token、密钥、认证头、证书、数据库连接串、对象存储配置、手机号、身份证号、内网地址或个人本地路径。
- 写入规则、文档、提交信息或总结时，对敏感信息脱敏；必要时改写为“项目内目录”“本地仓库配置”“用户级配置”等通用描述。
- 不主动执行部署、发布、远程上传、删除数据、修改共享基础设施等有外部副作用的操作，除非用户明确要求并给出目标环境。

## Git 规范

- 不主动执行 `git commit`；只有用户明确要求提交时才提交。
- 提交前必须查看实际 diff，提交说明结合改动内容，不写泛泛模板。
- 提交标题优先中文，可使用 conventional commit 前缀，例如 `feat: 新增火情任务状态同步`。
- 不在提交信息中写敏感信息、内网地址、账号、Token、密钥或完整异常堆栈。
