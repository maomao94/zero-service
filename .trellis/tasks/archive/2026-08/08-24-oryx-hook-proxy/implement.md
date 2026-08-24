# implement.md — oryxgtw / oryxserver

## 执行清单（有序）

1. **oryxgtw（HTTP 服务）**
   - [ ] `app/oryxgtw/oryxgtw.api`：定义 `HookRequest`（snake_case json tag 对齐 Oryx 回调）、`HookReply{Code int}`，`POST /v1/hook`。
   - [ ] `app/oryxgtw/gen.sh`：`goctl api format` + `goctl api go`。
   - [ ] `internal/config/config.go`：`rest.RestConf`（无 DB，最小）。
   - [ ] `internal/svc/servicecontext.go`：最小 `ServiceContext`。
   - [ ] `internal/logic/hook/`：`hook.go`（按 action 分发）+ 5 个 action 处理（onPublish/onUnpublish/onRecordBegin/onRecordEnd/onOcr），日志 + 返回 code 0。
   - [ ] `internal/handler/`：路由注册 `POST /v1/hook`。
   - [ ] `oryxgtw.go`：go-zero REST 入口（CORS 对齐 lalhook）。
   - [ ] `etc/oryxgtw.yaml`。

2. **oryxserver（gRPC 服务）**
   - [ ] `app/oryxserver/oryxserver.proto`：8 个 RPC + message，字段 snake_case + `json_name`，含 `java_package`/`java_multiple_files`/`java_outer_classname`。
   - [ ] `app/oryxserver/gen.sh`：`goctl rpc protoc`（对齐 lalproxy 的 protoc + validate + descriptor + openapiv2）。
   - [ ] `internal/config/config.go`：`zrpc.RpcServerConf` + `NacosConfig` + `OryxConfig{Ip,Port,Timeout,Token}`。
   - [ ] `internal/svc/servicecontext.go`：`OryxBaseUrl` + `OryxClient httpc.Service`（带 Bearer token）。
   - [ ] `internal/server/oryxserverserver.go`：注册 8 个 RPC。
   - [ ] `internal/logic/`：8 个 logic，参数校验 → httpc 调 Oryx → JSON 解析 → 响应。
   - [ ] `oryxserver.go`：zRPC 入口 + Nacos 注册 + grpcx interceptor。
   - [ ] `etc/oryxserver.yaml`。

3. **字段核对**：从 Oryx 源码核对 record/hooks API 精确请求/响应字段，修正 proto 与 logic。

4. **验证**：`go build ./...`、`go vet ./...`、`git diff --check`。

## Oryx API 清单（oryxserver 封装）

| RPC | HTTP | Oryx API |
|-----|------|----------|
| `Versions` | GET | `/terraform/v1/mgmt/versions` |
| `RecordQuery` | POST | `/terraform/v1/hooks/record/query` |
| `RecordApply` | POST | `/terraform/v1/hooks/record/apply` |
| `RecordEnd` | POST | `/terraform/v1/hooks/record/end` |
| `RecordFiles` | POST | `/terraform/v1/hooks/record/files` |
| `RecordRemove` | POST | `/terraform/v1/hooks/record/remove` |
| `HooksApply` | POST | `/terraform/v1/mgmt/hooks/apply` |
| `HooksQuery` | POST | `/terraform/v1/mgmt/hooks/query` |

## 验证命令

```bash
cd app/oryxgtw && ./gen.sh && cd ../..
cd app/oryxserver && ./gen.sh && cd ../..
go build ./...
go vet ./...
git diff --check
```

## 回滚点

- 每个服务独立，未 `go build` 通过前不提交；生成文件 diff 异常时回退 gen.sh 改动并重新生成。
- 字段核对若发现 Oryx API 与文档不一致，优先改 proto/logic，不回退已确认的回调设计。
