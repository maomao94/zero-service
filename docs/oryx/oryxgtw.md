# Oryx HTTP 回调网关（oryxgtw）

Oryx HTTP 回调的统一入口服务，接收 Oryx/SRS 的五类回调事件并按 `action` 字段分发处理。

## 服务信息

| 项目 | 值 |
| --- | --- |
| 服务名 | `oryxgtw` |
| 协议 | HTTP（go-zero REST） |
| 前缀 | `/v1/hook` |
| 超时 | 7200s |
| 中间件 | `OryxHookAuth` |

## 回调类型

Oryx 支持以下五类回调事件，均通过同一入口 `/v1/hook` 接收：

| action | 说明 | 处理逻辑 |
| --- | --- | --- |
| `on_publish` | 推流开始 | 记录日志，返回 `code=0` 放行推流 |
| `on_unpublish` | 推流结束 | 记录日志 |
| `on_record_begin` | 录制开始 | 记录日志 + 调用 `oryxserver.RecordBeginHook` 落库 |
| `on_record_end` | 录制结束 | 记录日志 + 调用 `oryxserver.RecordEndHook` 落库（含产物信息） |
| `on_ocr` | OCR 识别完成 | 记录日志 |

## 回调请求体

```json
{
  "request_id": "请求唯一 ID",
  "action": "on_publish",
  "opaque": "不透明标识",
  "vhost": "虚拟主机",
  "app": "应用名",
  "stream": "流名称",
  "param": "推流 URL 参数",
  "uuid": "录制/OCR 任务 UUID",
  "artifact_code": 0,
  "artifact_path": "录制文件路径",
  "artifact_url": "录制文件 URL",
  "prompt": "OCR prompt",
  "result": "OCR 结果"
}
```

## 回调响应

```json
{
  "code": 0
}
```

- `code=0`：成功，`on_publish` 需返回 0 才放行推流
- `code!=0`：失败

## 鉴权

通过 `OryxHookAuth` 中间件验证回调合法性。配置中的 `HookOpaque` 需与 Oryx `hooks/apply` 接口配置的 `opaque` 一致。

```yaml
# etc/oryxgtw.yaml
HookOpaque: "your-opaque-value"  # 为空则跳过校验
```

## 配置示例

```yaml
Name: oryxgtw
Host: 0.0.0.0
Port: 8080
HookOpaque: ""
OryxServerConf:
  Etcd:
    Hosts:
      - etcd:2379
    Key: oryxserver.rpc
```

## 数据流

```
Oryx/SRS ──HTTP POST──> oryxgtw
                            │
                            ├─ on_publish / on_unpublish → 日志
                            ├─ on_record_begin → gRPC RecordBeginHook → oryxserver
                            ├─ on_record_end   → gRPC RecordEndHook   → oryxserver
                            └─ on_ocr          → 日志
```

## 注意事项

1. 回调处理失败仅记录日志，不影响响应（始终返回 `code=0`），避免阻塞 Oryx 推流
2. `on_publish` 必须返回 `code=0`，否则 Oryx 会拒绝推流
3. 录制落库通过 gRPC 异步调用 `oryxserver`，两者解耦
