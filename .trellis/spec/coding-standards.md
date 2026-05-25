# 编码规范

> 本项目基于 Go / go-zero 开发，AI 相关业务基于 Eino。编码规范优先服务于稳定交付、最小变更和可验证结果。

## 技术栈

- 语言：Go 1.25+
- 微服务框架：go-zero
- RPC/API：gRPC、grpc-gateway、Protocol Buffers、go-zero `.api`
- AI 框架：CloudWeGo Eino / Eino ADK / MCP / OpenAI-compatible API
- 代码生成：goctl、protoc、各服务 `gen.sh`

## AI 协作纪律

- 默认使用中文沟通、提问、任务清单和总结；用户明确要求其他语言时再切换。
- 修改前先阅读相邻实现、配置、Trellis task 和相关 spec，不基于猜测改动。
- OpenCode 与 GoLand AI 规则分别放在 `.opencode/rules/**` 和 `.aiassistant/rules/**`，两边文件名和内容保持一致；不要把 OpenCode 规则追加到 `AGENTS.md` 的 Trellis 托管块后。
- `.opencode/agents/**`、`.opencode/skills/**`、`.opencode/commands/**`、`.opencode/plugins/**`、`.opencode/lib/**` 属于 Trellis/OpenCode 适配层，除非用户明确要求修改适配器，否则不要改。
- `.aiassistant/rules/**` 与 `.opencode/rules/**` 内容保持同步，分别供 GoLand AI 和 OpenCode 读取；修改时两边同步更新。
- 能通过搜索代码和读取配置推断的低风险事项自主推进；意图多解且影响大、不可逆操作、缺关键参数或规则冲突时再提问。
- 按最小影响范围修改，避免无关重构、无关格式化和大范围重排。
- 不主动创建额外文档或注释；但 `.api`、`.proto`、导出公共能力和复杂协议字段必须保留必要说明。
- 总结时说明改动内容、影响范围、已执行验证和未验证原因。

## AI 工具诊断配置

### 1. Scope / Trigger

- Trigger: 当 `lsp_diagnostics` 对项目内文档、配置或脚本提示 `No LSP server configured for extension`，先补语言服务器和 OpenCode LSP 绑定，再把诊断作为验证手段使用。
- Scope: 只记录可复用的命令、配置结构和验证方式；不要把个人绝对路径、API Key、Token 或本机账号写进 spec、规则、提交信息或总结。

### 2. Signatures

- Markdown LSP: `marksman server`
- Homebrew 安装：`brew install marksman`
- OpenCode 配置根节点：`lsp.<serverName>.command` + `lsp.<serverName>.extensions`

### 3. Contracts

```jsonc
{
  "lsp": {
    "marksman": {
      "command": ["marksman", "server"],
      "extensions": [".md"]
    }
  }
}
```

- `command` 必须是可在当前 shell `PATH` 中解析的命令和参数数组。
- `extensions` 使用带点扩展名，例如 `.md`。
- 优先把通用 LSP 绑定写入 OpenCode `opencode.json(c)` 的根级 `lsp`；不要写入旧 OhMyOpenAgent 插件私有配置的顶层 `lsp`。
- 如果只配置 OhMyOpenAgent 的 LSP MCP 项目级 server map，使用项目内 `.opencode/lsp.json`，不要和 OpenCode 原生 `opencode.json(c)` 配置混用。
- 优先使用 Homebrew 安装本地工具；只有 Homebrew 无可用公式或用户明确同意时，再考虑 npm 全局安装。

### 4. Validation & Error Matrix

| 条件 | 处理 |
|------|------|
| `command -v marksman` 为空 | 先通过 Homebrew 安装 `marksman` |
| `marksman --version` 失败 | 修复安装或 PATH，不写 OpenCode 配置 |
| `lsp_diagnostics` 仍提示 `.md` 未配置 | 检查 OpenCode 配置是否有根级 `lsp.marksman` |
| OpenCode 启动或 `opencode --help` 失败 | 回退刚加入的 JSONC 配置并修正语法 |
| Markdown 跨文件引用/补全异常 | 确认项目是 Git 仓库，或按 Marksman 规则补 `.marksman.toml` |
| spec/总结需要描述本机配置 | 使用“用户级配置”这类泛称，不写个人绝对路径或密钥 |

### 5. Good/Base/Bad Cases

- Good: `marksman --version` 成功，`lsp_diagnostics` 对 `.md` 返回真实诊断或 `No diagnostics found`。
- Base: `.md` 文件没有 Markdown LSP 时，用 `git diff --check` 做最低限度格式验证，并说明未执行 LSP 的原因。
- Bad: 工具已安装但未绑定 OpenCode，仍把 `No LSP server configured` 当作文件无问题。

### 6. Tests Required

- 安装验证：执行 `marksman --version`。
- 配置验证：执行 `opencode --help` 或同等只读命令，确认配置可解析。
- 诊断验证：对一个项目内 `.md` 文件执行 `lsp_diagnostics`，断言不再出现 `No LSP server configured for extension: .md`。

### 7. Wrong vs Correct

#### Wrong

```jsonc
{
  "lsp": {
    "marksman": {
      "command": ["marksman"],
      "extensions": ["md"]
    }
  }
}
```

#### Correct

```jsonc
{
  "lsp": {
    "marksman": {
      "command": ["marksman", "server"],
      "extensions": [".md"]
    }
  }
}
```

## 代码生成流程

每个业务服务通常提供 `gen.sh` 脚本用于基础代码生成：

- 网关接口：先修改 `.api` 文件定义接口，再执行对应 `gen.sh`。
- gRPC 服务：先修改 `.proto` 文件定义服务，再执行对应 `gen.sh`。
- 生成后再进入 `internal/logic/` 编写业务逻辑，并检查生成代码 diff。

禁止跳过 `gen.sh` 直接手写 Handler、Server、Routes、Types 或 pb 代码。需要调整生成结果时，优先修改源 `.api` / `.proto` 或生成模板。

## 命名约定

### API 网关

- 请求结构命名：`xxxRequest`
- 响应结构命名：`xxxResponse`
- 请求和响应必须成对出现，并保持注释完整。

### gRPC

- 请求结构命名：`xxxReq`
- 响应结构命名：`xxxRes`
- 请求和响应必须成对出现，例如 `chatReq` + `chatRes`。

## 编码规则

- 遵循 Go / go-zero / Google 开发规范，禁止 Java 风格命名、异常处理和过度封装。
- Handler/Server 负责参数接收、校验、调用 Logic 和返回结果；业务编排放在 Logic。
- 跨服务复用能力沉淀到 `common/`，服务内部有状态逻辑保留在对应服务 `internal/`。
- 新增依赖前先检查 `go.mod`、相邻模块和现有封装，不重复引入功能相近的库。
- 涉及数据库、Redis、消息队列、MQTT、OSS、Docker、Eino 或 DJI Cloud API 时，优先复用已有 model、client、cache、config、SDK 和 `common/` 封装。
- 工具函数、复杂协议转换和关键业务分支应有单元测试；gRPC Logic 以业务编排为主非必要不写单测，生成代码非必要不写自定义测试。
- `.api` / `.proto` 注释必须与实现行为保持一致，API 转 gRPC 时注释语义保持一致。

## 安全规则

- 不新增、打印或记录明文密码、Token、密钥、认证头、证书、数据库连接串、对象存储配置、手机号、身份证号、内网地址或个人本地路径。
- 写入规则、文档、提交信息或总结时，对敏感信息脱敏；必要时改写为“项目内目录”“本地仓库配置”“用户级配置”等通用描述。
- 不主动执行部署、发布、远程上传、删除数据、修改共享基础设施等有外部副作用的操作；除非用户明确要求并给出目标环境。

## Git 规范

- 不主动执行 `git commit`；只有用户明确要求提交时才提交。
- 提交前必须查看实际 diff，提交说明结合改动内容，不写泛泛模板。
- 提交标题优先中文，可使用 conventional commit 前缀，例如 `feat: 新增火情任务状态同步`。
- 不在提交信息中写敏感信息、内网地址、账号、Token、密钥或完整异常堆栈。
