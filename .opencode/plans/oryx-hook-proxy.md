# 计划：基于 Oryx 的 hook 与 gRPC 代理服务（最小实现单元）

## Goal

为业务侧提供对接 Oryx 的最小基础服务层，参照本仓库 `app/lalhook` 与 `app/lalproxy` 的既有模式：

1. **`oryxhook`（HTTP 服务）**：接收 Oryx 的 HTTP Callback，字段与 Oryx 保持一致，正确响应 `{"code":0}`。
2. **`oryxproxy`（gRPC 服务）**：把 Oryx 的 HTTP OpenAPI 封装为 gRPC，注册 Nacos，供 Java 服务调用：`Java → Nacos → oryxproxy(gRPC) → Oryx(HTTP)`。

本阶段为最小实现单元，**暂不处理工单关联与落库**（后续扩展）。

## In Scope

### oryxhook（HTTP 服务，go-zero REST）
接收 Oryx 五类回调，字段与 Oryx 官方 JSON 完全一致：

| 回调 | Oryx 字段 |
|------|-----------|
| `on_publish` | `request_id / action / opaque / vhost / app / stream / param` |
| `on_unpublish` | `request_id / action / opaque / vhost / app / stream` |
| `on_record_begin` | 上同 + `uuid` |
| `on_record_end` | 上同 + `uuid / artifact_code / artifact_path / artifact_url` |
| `on_ocr` | `uuid / prompt / result`（含公共字段） |

- 统一端点 `POST /v1/hook`，按 `action` 字段分发（Oryx 回调是 action 驱动，target URL 为单一地址）。
- 响应：统一返回 `{"code":0}`（`on_publish` 据此放行，其余事件忽略响应）。
- 处理逻辑：日志记录 + 空实现 stub（预留后续落库/转发）。

### oryxproxy（gRPC 服务，go-zero zRPC）
封装 Oryx 基础 HTTP API（聚焦录制 + 回调配置 + 基础信息），字段与 Oryx 保持一致：

| gRPC RPC | Oryx HTTP API |
|----------|---------------|
| `Versions` | `/terraform/v1/mgmt/versions` |
| `RecordQuery` | `/terraform/v1/hooks/record/query` |
| `RecordApply` | `/terraform/v1/hooks/record/apply` |
| `RecordEnd` | `/terraform/v1/hooks/record/end` |
| `RecordFiles` | `/terraform/v1/hooks/record/files` |
| `RecordRemove` | `/terraform/v1/hooks/record/remove` |
| `HooksApply` | `/terraform/v1/mgmt/hooks/apply` |
| `HooksQuery` | `/terraform/v1/mgmt/hooks/query` |

- 沿用 lalproxy 模式：`oryxproxy.proto`（含 `java_package`/`java_multiple_files`）、`httpc` 调上游、`nacosx` 注册 Nacos、`extproto` 错误码。
- 鉴权：Bearer token 从配置读取（`OryxConfig.Token`），请求头 `Authorization: Bearer <token>`。
- 命名/端口暂沿用 lal 风格：`oryxhook` 端口 11003，`oryxproxy` 端口 21003（实施时可在 etc/yaml 调整）。

## Out of Scope（本阶段不做）

- 工单号关联（流名/param/opaque 编码、映射表）。
- hook 服务 DB 落库（录制事件、artifact 持久化）。
- SRS 代理 API（`/api/v1/...`）与 Oryx 全量 OpenAPI（转码/AI/虚拟直播等）。
- 录制 glob 动态启停的工单化封装（本期只提供 record/hooks 基础 RPC）。

## Acceptance Criteria

- [ ] `oryxhook` 启动，`POST /v1/hook` 能接收并正确解析 Oryx 五类回调，返回 `{"code":0}`。
- [ ] `oryxhook` 回调字段与 Oryx 官方 JSON 字段一一对应（snake_case 对齐）。
- [ ] `oryxproxy` 启动并注册 Nacos，上述 8 个 RPC 可被调用并正确转发到 Oryx HTTP API。
- [ ] `oryxproxy` 携带 Bearer token 鉴权，token 从配置注入。
- [ ] proto 含 Java 相关 option，可生成 Java 端调用桩。
- [ ] `go build ./...` 通过；`go vet ./...` 通过（如项目配置了 lint）。

## Key Decisions（已确认）

1. 范围：聚焦录制 + 回调配置 + 基础信息的**最小集**，不做全量 OpenAPI。
2. 字段：尽量与 Oryx 官方字段保持一致（snake_case）。
3. 本阶段不做工单关联、不做落库。

## Risks / Deferred

- **字段精确值**：Oryx 各 API 的请求/响应 JSON 精确字段需在实施时从 Oryx 源码（`platform/` 下对应 handler）或 `系统配置 > OpenAPI` 核对，避免 proto 字段偏差。
- **回调端点形态**：Oryx target URL 配置为单一 URL + `action` 分发；若实际是 per-action URL，则改为分端点（实施时用 `hooks/query` 核对）。
- **鉴权 token 获取**：需运维提供或从 Oryx OpenAPI 页获取，写入 `etc/oryxproxy.yaml`。
- 工单关联、落库、全量 API 封装作为后续迭代。

## 实施步骤（implement 概要）

1. `app/oryxhook`：`oryxhook.api`（types：五类回调 struct + EmptyReply/CodeReply）→ `goctl api` 生成 → 实现 handler/logic（按 action 分发 + 日志 + 返回 code）。
2. `app/oryxproxy`：`oryxproxy.proto`（8 个 RPC + message，字段对齐 Oryx）→ `goctl rpc protoc` 生成 → config（OryxConfig: Ip/Port/Timeout/Token + NacosConfig）→ svc（httpc + baseUrl + token）→ logic 逐个封装。
3. 核对 Oryx 字段后修正 proto/api 类型。
4. `go build ./...` + 冒烟（本地起 Oryx 或 mock 验证回调与 RPC 转发）。
