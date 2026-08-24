# 基于 Oryx 的 HTTP 网关与 gRPC 服务（最小实现单元）

## Goal

为业务（巡检工单驱动的"按需拉流录制"）提供对接 Oryx 的最小基础服务层：

1. **`oryxgtw`（HTTP 服务）**：接收 Oryx 的 HTTP Callback，字段与 Oryx 保持一致，正确响应 `{"code":0}`；同时作为对外的 HTTP 网关。
2. **`oryxserver`（gRPC 服务）**：把 Oryx 的 HTTP OpenAPI 封装为 gRPC，注册 Nacos，供 Java 服务调用：`Java → Nacos → oryxserver(gRPC) → Oryx(HTTP)`。

参考实现：`app/lalhook`（HTTP 回调模式）、`app/lalproxy`（gRPC 代理 + Nacos 模式）。Oryx 文档以 **6.0 (Stable)** 为准。

本阶段为最小实现单元，**暂不处理工单关联与落库**（后续扩展）。

## Confirmed Facts（已核实，Oryx v6）

### Oryx 回调体系（区别于 lal / SRS）
- Oryx 的"HTTP Callback"是其自定义的一层回调，通过 `/terraform/v1/mgmt/hooks/apply` 配置 target URL；SRS 层 `http_hooks` 默认指向 Oryx 内部 `/terraform/v1/hooks/srs/verify`、`/hls`，由 Oryx 再转发。
- 回调事件与字段（snake_case）：
  - `on_publish`：`request_id / action / opaque / vhost / app / stream / param`。响应 `{"code":0}` 放行，否则拒绝推流。
  - `on_unpublish`：`request_id / action / opaque / vhost / app / stream`，忽略响应。
  - `on_record_begin`：公共字段 + `uuid`，忽略响应。
  - `on_record_end`：公共字段 + `uuid / artifact_code / artifact_path / artifact_url`，忽略响应。
  - `on_ocr`：公共字段 + `uuid / prompt / result`，忽略响应。
- 请求 `Content-Type: application-json`，成功响应统一 `200 + {"code":0}`。

### Oryx HTTP API 分层（Bearer/JWT 鉴权，token 取自 OpenAPI 页，密码 `MGMT_PASSWORD`）
- 自有 OpenAPI `/terraform/v1/...`：
  - 录制：`/terraform/v1/hooks/record/apply|query|globs|post-processing|remove|end|files`
  - 回调配置：`/terraform/v1/mgmt/hooks/apply|query`
  - 流管理：`/terraform/v1/mgmt/streams/query|kickoff`
  - 版本：`/terraform/v1/mgmt/versions`
- 代理 SRS API `/api/v1/...`（本阶段不封装）。

### 关键约束：Oryx 录制是"pattern 驱动"，非"按条流启停"
- 录制任务 `uuid` 在 `on_record_begin` 下发；流停推后用 `/terraform/v1/hooks/record/end` 快速生成 mp4。

## Requirements

- R1：新建 `oryxgtw` HTTP 服务，接收 Oryx 五类回调（统一 `POST /v1/hook`，按 `action` 分发），正确响应 `{"code":0}`。
- R2：新建 `oryxserver` gRPC 服务，封装 Oryx 录制 + 回调配置 + 基础信息 API 为 gRPC，注册 Nacos。
- R3：`oryxserver` 支持 Oryx Bearer token 鉴权，token 从配置注入。
- R4：proto 含 Java 相关 option，可生成 Java 调用桩。
- R5：回调字段与 Oryx 官方 JSON 字段一一对应（snake_case）。

## Acceptance Criteria

- [ ] `oryxgtw` 启动，`POST /v1/hook` 能接收并正确解析 Oryx 五类回调，返回 `{"code":0}`。
- [ ] `oryxgtw` 回调字段与 Oryx 官方 JSON 字段一一对应。
- [ ] `oryxserver` 启动并注册 Nacos，`Versions` + 录制 5 个 RPC + 回调配置 2 个 RPC 可被调用并正确转发到 Oryx HTTP API。
- [ ] `oryxserver` 携带 Bearer token 鉴权，token 从配置注入。
- [ ] proto 含 Java option，可生成 Java 端调用桩。
- [ ] `go build ./...`、`go vet ./...` 通过。

## Out of Scope（本阶段不做）

- 工单号关联（流名/param/opaque 编码、映射表）。
- hook 服务 DB 落库（录制事件、artifact 持久化）。
- SRS 代理 API（`/api/v1/...`）与 Oryx 全量 OpenAPI（转码/AI/虚拟直播等）。
- 录制 glob 动态启停的工单化封装。
- gRPC 服务的 HTTP 网关（grpc-gateway）——如需后续加。
