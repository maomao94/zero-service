# 设计 - 迁移废弃 API (SA1019, 18处)

## 迁移映射

### 1. bubbles viewport（4处）
| 旧 API | 新 API |
|---|---|
| `viewport.LineUp(n)` | `viewport.ScrollUp(n)` |
| `viewport.LineDown(n)` | `viewport.ScrollDown(n)` |

新方法返回 `[]string`，调用处均忽略返回值，直接替换即可。

文件：
- cli/uix/components/logviewer.go:119,124
- cli/uix/timeline.go:88,89

### 2. bubbles filepicker（3处）
| 旧 API | 新 API |
|---|---|
| `fp.Height = n`（写） | `fp.SetHeight(n)` |
| `fp.fp.Height`（读，无 getter） | 包装层自持字段跟踪 |

`SetHeight` 内部就是写 `m.Height` 并刷新 `max`，写路径直接换。

具体位置（当前行号）：
- filepicker.go:34 `fp.Height = 8` → `fp.SetHeight(8)`
- filepicker.go:105 `fp.fp.Height = height-4` → `fp.fp.SetHeight(height-4)`
- filepicker.go:110 `Height()` 读取 `fp.fp.Height`

读路径：`FilePicker.Height()` 返回 `fp.fp.Height + 4`。因 `filepicker.Model` 无 `Height()` getter，
包装层已持有 `height` 字段（line 17），语义对齐：`SetSize` 里 `fp.height = height` 与
`fp.fp.SetHeight(height-4)` 同步，`Height()` 改为返回 `fp.height`。但注意默认构造
`NewFilePicker` 里 `height: 24` 与 `fp.SetHeight(8)` 不一致（24 vs 12）。保守做法：
`Height()` 返回 `fp.fp 的高度`无法直接读，因此让 `NewFilePicker` 同步 `fp.height = 8`，保持
`Height()` 语义 = 列表高度 + 4。

### 3. eino adk.AgentMiddleware（5处）→ 删除
无调用方，整体删除：
- common/einox/agent/agent_option.go:
  - `options.middlewares []adk.AgentMiddleware` 字段
  - `WithMiddleware` / `WithMiddlewares` 函数及注释
- common/einox/agent/agent.go:120-122 的 `agentCfg.Middlewares` 赋值块
- common/einox/agent/factory.go:
  - :67-69 的 `deepCfg.Middlewares` 赋值块
  - :203 的 `cc.middlewares` 拷贝
  - `needsWorkflowCoordinator`（:187）中 `len(cfg.middlewares) > 0` 判断

### 4. mcpx 协议废弃字段（6处）→ 删除透传
common/mcpx/client.go `buildClientOptions`：
- 删除 :213-217 CreateMessageHandler 透传块
- 删除 :261-265 LoggingMessageHandler 透传块

## 验证

- `staticcheck -checks=SA1019 ./...` 输出为 0
- `go build ./...`
- `go vet ./...`
- 若 cli/uix 有相关测试则运行

## 不做的事

- 不修改 viewport/filepicker 滚动与高度显示语义
- 不引入新依赖
- 不处理 x/net/context（go fix 已迁移）
