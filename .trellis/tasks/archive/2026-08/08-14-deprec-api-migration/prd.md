# 迁移废弃 API (SA1019, 18处)

## Goal

将 staticcheck `SA1019` 报出的 18 处废弃 API 用法全部迁移到新 API，消除构建告警。
（x/net/context 的 2 处已被 `go fix` 自动迁移为标准库 `context`，不在本任务范围。）

## 背景

项目升级到 go1.26 后，用 `staticcheck -checks=SA1019 ./...` 扫描出旧 API 用法。
`go fix` 已自动处理了一批语言/标准库新特性（`reflect.Pointer`、`strings.FieldsSeq`、
`strings.Builder` 等），剩余 18 处属于第三方库废弃 API，需手动迁移。

## Requirements

1. **charmbracelet/bubbles viewport**（cli/uix/components/logviewer.go、cli/uix/timeline.go，共4处）
   - `viewport.LineUp(n)` → `viewport.ScrollUp(n)`
   - `viewport.LineDown(n)` → `viewport.ScrollDown(n)`
   - 新方法返回 `[]string`，当前调用均丢弃返回值，无行为影响。

2. **charmbracelet/bubbles filepicker**（cli/uix/components/filepicker.go，共3处）
   - `fp.Height = 8` → `fp.SetHeight(8)`
   - `fp.fp.Height = height-4` → `fp.fp.SetHeight(height-4)`
   - `Height()` 方法读取 `fp.fp.Height` → 需用包装层自有字段跟踪列表高度，保持返回值语义不变。

3. **cloudwego/eino adk.AgentMiddleware**（common/einox/agent/，共5处）
   - 结构体式 `adk.AgentMiddleware`（config.Middlewares）废弃，替换为接口式 `adk.ChatModelAgentMiddleware`（config.Handlers）。
   - 当前项目内 `WithMiddleware`/`WithMiddlewares` 无任何调用方。

4. **modelcontextprotocol go-sdk 协议废弃字段**（common/mcpx/client.go，共6处）
   - `mcp.ClientOptions.CreateMessageHandler`（sampling）与 `LoggingMessageHandler`（logging）随协议 2026-07-28 (SEP-2577) 废弃。
   - 当前仅透传拷贝，无调用方设置这两个字段。

## Constraints

- 迁移后 `staticcheck -checks=SA1019 ./...` 必须清零。
- `go build ./...` 必须通过。
- 不改变运行时行为：viewport/filepicker 的滚动、高度显示语义与迁移前一致。
- 迁移涉及公共 API 删除的部分须先与用户确认。

## Acceptance Criteria

- [ ] staticcheck SA1019 报告为 0。
- [ ] go build ./... 通过。
- [ ] go vet ./... 通过。
- [ ] cli/uix 相关组件行为不变（有测试则运行）。
- [ ] common/mcpx、common/einox/agent 编译通过且功能语义不变。

## Notes

- 决策点：eino AgentMiddleware 采用「删除」还是「适配器保留 API」；mcpx 采用「删除透传」还是「保留并抑制告警」——见 design.md。
