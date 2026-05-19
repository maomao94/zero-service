# AI Solo 本地 Smoke Checklist

本文用于本地验收 `aiapp/aisolo` 和 `aiapp/aigtw` 的最小可交付链路。它不替代单元测试；目标是在有或没有真实模型 Key 的情况下，快速判断服务启动、健康状态、SSE 边界和知识库配置是否符合预期。

## 范围

- `aiapp/aisolo`：Eino Agent/Solo gRPC 服务，默认监听 `0.0.0.0:23002`。
- `aiapp/aigtw`：HTTP 网关和 Solo Web UI，默认监听 `0.0.0.0:13001`。
- `common/einox`：runtime runner、工具、知识库和协议事件的公共能力。

不覆盖真实生产部署、Nacos、容器编排、外部向量库可用性或真实模型质量评估。

## 前置条件

1. Go 版本满足项目根 README 要求。
2. 当前目录位于项目根目录。
3. `aigtw` 必须配置 JWT secret。推荐用环境变量，不要写入仓库配置：

```bash
export AIGTW_JWT_ACCESS_SECRET='local-dev-secret-change-me'
```

4. `aisolo` 的 `Model.APIKey` 可以为空。为空时服务应能启动，但 Agent 执行能力在 health dependencies 中表现为模型/runner 不可用或缺失。
5. 如果需要真实 Agent 回复，配置 `aiapp/aisolo/etc/aisolo.yaml` 的 `Model.APIKey`，或按服务支持的环境变量注入模型 Key。

## 启动

终端 A：启动 `aisolo`。

```bash
go run ./aiapp/aisolo -f aiapp/aisolo/etc/aisolo.yaml
```

无模型 Key 时，应看到类似提示：

```text
model api key is empty; Agent execution stays unavailable until Model.APIKey or AISOLO_MODEL_API_KEY is configured
```

终端 B：启动 `aigtw`。

```bash
go run ./aiapp/aigtw -f aiapp/aigtw/etc/aigtw.yaml
```

如果没有配置 `AIGTW_JWT_ACCESS_SECRET` 或 `JwtAuth.AccessSecret`，`aigtw` 应拒绝启动并输出：

```text
jwt access secret is empty; set JwtAuth.AccessSecret or AIGTW_JWT_ACCESS_SECRET
```

## Health 和 Meta

`aigtw` health 不需要 JWT：

```bash
curl -s http://127.0.0.1:13001/health
```

期望：

- `status` 为 `ok`。
- `version` 为 `aigtw`。
- `ready` 反映核心依赖状态。
- `dependencies.jwt` 为 `ok`。
- `dependencies.aisolo_rpc` 为 `ok` 表示网关已构造 aisolo RPC client。
- `dependencies.knowledge` 为 `disabled`、`ok` 或 `misconfigured`。

Solo meta 需要 JWT：

```bash
TOKEN='<local-jwt>'
curl -s -H "Authorization: Bearer ${TOKEN}" http://127.0.0.1:13001/solo/v1/meta
```

期望返回：

- `ready`。
- `dependencies`。
- 兼容字段 `knowledgeBackend`、`knowledge`、`knowledge_error`。

## Solo HTTP 基础链路

以下接口都需要 `Authorization: Bearer <token>`，且 JWT claim 需能映射出用户 ID。默认配置的 `JwtAuth.ClaimMapping` 支持把 `user-id` 映射为内部 `user_id`。

列出模式：

```bash
curl -s -H "Authorization: Bearer ${TOKEN}" http://127.0.0.1:13001/solo/v1/modes
```

创建会话：

```bash
curl -s -X POST http://127.0.0.1:13001/solo/v1/sessions \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"title":"local smoke","mode":"agent","uiLang":"zh"}'
```

记录响应里的 `sessionId`：

```bash
SESSION_ID='<session-id>'
```

获取会话和历史消息：

```bash
curl -s -H "Authorization: Bearer ${TOKEN}" "http://127.0.0.1:13001/solo/v1/sessions/${SESSION_ID}"
curl -s -H "Authorization: Bearer ${TOKEN}" "http://127.0.0.1:13001/solo/v1/sessions/${SESSION_ID}/messages?limit=20"
```

## SSE Chat

发起流式对话：

```bash
curl -N -X POST http://127.0.0.1:13001/solo/v1/chat \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"sessionId\":\"${SESSION_ID}\",\"message\":\"用一句话介绍你自己\",\"mode\":\"agent\",\"uiLang\":\"zh\"}"
```

期望：

- HTTP 响应为 `Content-Type: text/event-stream`。
- 每一帧是 `data: <json>`，以空行分隔。
- JSON 内容是 Solo protocol event，不应被二次 JSON marshal。
- 最后一帧应包含最终事件，服务随后结束流。

无真实模型 Key 时，该链路可能返回 executor/model unavailable 类错误；这属于预期限制，但服务不应崩溃。

非法请求应在打开 SSE 前失败。例如空消息：

```bash
curl -i -X POST http://127.0.0.1:13001/solo/v1/chat \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"sessionId\":\"${SESSION_ID}\",\"message\":\"   \"}"
```

期望不是 `200 OK`，且响应头不应是 `text/event-stream`。

## Resume 中断恢复

当模型触发中断事件时，先查询中断详情：

```bash
INTERRUPT_ID='<interrupt-id>'
curl -s -H "Authorization: Bearer ${TOKEN}" "http://127.0.0.1:13001/solo/v1/interrupt/${INTERRUPT_ID}"
```

恢复：

```bash
curl -N -X POST "http://127.0.0.1:13001/solo/v1/interrupt/${INTERRUPT_ID}/resume" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"sessionId\":\"${SESSION_ID}\",\"action\":\"yes\",\"reason\":\"approved in local smoke\"}"
```

期望与 Chat 一致：SSE `data: <json>` 帧，final 后结束。

非法 action 应在打开 SSE 前失败：

```bash
curl -i -X POST "http://127.0.0.1:13001/solo/v1/interrupt/${INTERRUPT_ID}/resume" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"sessionId\":\"${SESSION_ID}\",\"action\":\"maybe\"}"
```

期望不是 `200 OK`，且响应头不应是 `text/event-stream`。

## 知识库 Smoke

默认 `Knowledge.Enabled: false`，知识库接口应返回 `knowledge is disabled`。启用时需要同时考虑：

- `aigtw` 和 `aisolo` 若要共享索引，必须配置同一后端：`gorm` 同 DataDir/DSN，或相同 Redis/Milvus 实例。
- `memory` backend 仅进程内有效，不适合跨 `aigtw` 和 `aisolo` 共享。
- Embedding `api_key` 和 `model` 需要真实可用配置。

创建知识库：

```bash
curl -s -X POST http://127.0.0.1:13001/solo/v1/knowledge/bases \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"name":"local-smoke-kb"}'
```

入库和查询：

```bash
BASE_ID='<base-id>'
curl -s -X POST "http://127.0.0.1:13001/solo/v1/knowledge/bases/${BASE_ID}/ingest" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"filename":"hello.md","content":"Eino is the AI application framework used by this project."}'

curl -s -X POST "http://127.0.0.1:13001/solo/v1/knowledge/bases/${BASE_ID}/query" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"query":"What framework is used?","topK":3}'
```

绑定会话知识库：

```bash
curl -s -X POST "http://127.0.0.1:13001/solo/v1/sessions/${SESSION_ID}/knowledge" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"knowledgeBaseId\":\"${BASE_ID}\",\"knowledgeBaseName\":\"local-smoke-kb\"}"
```

## 自动化验证

本地提交前至少执行：

```bash
go test ./aiapp/aigtw/... -count=1
go test ./aiapp/aisolo/... -count=1
go test ./common/einox/... -count=1
go vet ./aiapp/aigtw/... ./aiapp/aisolo/... ./common/einox/...
git diff --check -- aiapp/aigtw aiapp/aisolo common/einox docs/ai-solo-smoke-checklist.md
```

敏感信息扫描建议：

```bash
rg -n 'eyJhbGci' aiapp/aigtw aiapp/aisolo common/einox
rg -n 'AccessSecret:' aiapp/aigtw aiapp/aisolo common/einox | rg -v 'AccessSecret: ""'
rg -n 'api_key:' aiapp/aigtw aiapp/aisolo common/einox | rg -v 'api_key: ""'
```

## 当前仍需人工验收

- 真实模型 Key 下的 `aigtw -> aisolo -> einox runtime` 端到端回复质量。
- 真实 SSE 客户端或浏览器 UI 下的断线、刷新、中断恢复体验。
- 真实共享知识库后端下的上传、索引、检索、绑定、Agent 工具调用链路。
- 生产 JWT issuer、claim mapping、过期时间和权限隔离策略。
