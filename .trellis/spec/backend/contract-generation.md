# 契约与生成规范

## 适用范围

修改 RPC/HTTP 方法、字段、校验、Swagger、事件入口或任何生成文件时读取。

## 契约源

- gRPC 以服务目录的 `.proto` 为源；HTTP/BFF 以 `.api` 为源。生成的 `*.pb.go`、`*_grpc.pb.go`、`*.pb.validate.go`、Server、Handler、Routes 和 Types 不是编辑入口。
- 使用目标服务根目录的 `gen.sh`。脚本可能包含 validate、grpc-gateway、OpenAPI 或不同 goctl 参数，不能用一条全仓通用命令替代。
- 外部共享 proto 位于 `third_party/`；修改前确认它是上游同步还是本项目所有的契约。
- `.proto` / `.api` 注释、validation、HTTP annotation 和实现行为要一起审查，不能只让代码编译。

依据：`app/trigger/trigger.proto`、`app/trigger/gen.sh`、`app/bridgegtw/bridgegtw.api`、`app/bridgegtw/gen.sh`、`aiapp/aisolo/aisolo.proto`、`aiapp/aisolo/gen.sh`。

## Proto 字段命名规范

- **字段名**：统一使用 `snake_case`（如 `trigger_time`、`session_id`）。
- **JSON 兼容**：对外 JSON 保持 `camelCase`，通过 `[json_name = "camelCaseName"]` 标注：

```protobuf
message PlanPb {
  string plan_id = 1 [json_name = "planId"];
  string trigger_time = 2 [json_name = "triggerTime"];
}
```

- **关键属性**：`snake_case` → `CamelCase` 与 `camelCase` → `CamelCase` 生成相同的 Go struct 字段名，因此 `msg_id` 和 `msgId` 都生成 `MsgId`，Go 调用方无需修改。

## Proto JSON 序列化规则

- Proto 类型 **必须** 使用 `google.golang.org/protobuf/encoding/protojson`，**禁止** 使用 `encoding/json` 或 `jsonx`：

```go
import "google.golang.org/protobuf/encoding/protojson"

ruleJSON, err := protojson.Marshal(in.Rule)      // 正确
err = protojson.Unmarshal([]byte(pbJSON), &rule)  // 正确
```

- **原因**：`encoding/json` 使用 Go struct 的 `json` tag，`protojson` 使用 proto 的 `json_name` annotation。混用会导致 JSON 字段名不一致。
- 非 proto 类型（Go struct、map、slice 等）仍使用 `encoding/json` 或 `jsonx`。

### json.RawMessage 透传模式

当 proto JSON 需要嵌入 Go struct 序列化链路时，使用 `json.RawMessage` 作为透明字节载体：

```go
type CronJobExtra struct {
    Rule     json.RawMessage `json:"rule"`      // protojson 数据透传
    BizExtra json.RawMessage `json:"biz_extra"` // 业务 JSON
}
```

**流程**：
```
序列化:   PlanRulePb → protojson.Marshal → json.RawMessage(Rule) → json.Marshal(CronJobExtra) → 存储
反序列化: 存储 → json.Unmarshal(CronJobExtra) → json.RawMessage(Rule) → protojson.Unmarshal → PlanRulePb
```

`json.Marshal` 写入 `json.RawMessage` 时原样输出字节，protojson 编码的 JSON 完整透传。

### extproto 自定义 Option 引用

修改 `extproto.proto` 中被自定义 option 引用的字段时，**必须同时更新所有引用处**：

```protobuf
// 定义
int32 http_code = 5002 [json_name = "httpCode"];

// 引用 — 使用 proto 字段名，不是 json_name
_1_00_UNKNOWN = 100999 [(name) = "未知错误", (http_code) = 500];
```

`(httpCode)` 依赖于 proto 字段名 `httpCode`；重命名为 `http_code` 后引用必须改为 `(http_code)`。

## 兼容性规则

- Proto 已发布字段号不得复用；删除字段时评估 `reserved`，新增字段默认按旧客户端缺省值可工作。
- 不因 Go 命名偏好改变外部 JSON 字段、Socket.IO 事件、MQTT topic、method 或业务枚举。
- 区分缺省值与业务空值，尤其是时间、状态、分页游标和可选过滤条件；不要用魔法值代替明确语义。
- 修改请求/响应字段后搜索生成 client、网关、消息消费者、前端/外部文档与测试，不能只改服务端。
- 对第三方协议保持原始结构，在适配层转换为项目类型，不让上游变化扩散到业务层。

## 生成流程

1. 修改源契约并补充注释/校验。
2. 执行目标服务 `gen.sh`。
3. 审查所有生成文件，确认没有插件版本或工作目录造成的意外重排。
4. 实现或调整 Logic、适配器和调用方。
5. 运行目标服务与直接消费者测试。

## 反模式

- 直接修 `*.pb.go`、生成 Server 或 API Types。
- 为保持旧调用方编译而偷偷改变服务端语义，不更新契约说明。
- 在 Logic 手工拼 topic、method 或 JSON 字段，绕过协议包常量和类型。
- 生成后不检查 diff，提交无关版本噪声。
- **用 `encoding/json` 序列化 proto 类型**——必须用 `protojson`。
- **Proto 字段使用 camelCase**——必须用 `snake_case` + `json_name`。

## 验证

```bash
cd <service-directory>
./gen.sh
go test ./...
git diff --check
```

还要用 `rg` 搜索变更的字段名、RPC、事件名或枚举在仓库中的所有消费者。

## Proto 迁移检查清单

新增或修改 proto 字段时验证：

```bash
# 1. 检查 camelCase 字段（应在消息体内，非 enum/option/注释行）
grep -n '^\s*\(repeated\s\+\|map<[^>]*>\s\+\)\?\(bool\|int32\|int64\|uint32\|uint64\|float\|double\|string\|bytes\|[A-Z][A-Za-z0-9]*\)\s\+[a-zA-Z]*[A-Z][a-zA-Z]*\s*=' *.proto

# 2. 确认 json_name 标注存在且值为原驼峰名
grep -c 'json_name' *.proto

# 3. 搜索 encoding/json 用于 proto 类型（应在 Go 文件中）
rg 'json\.(Marshal|Unmarshal)' --glob '*.go' | grep -v '_test\.go'
```
