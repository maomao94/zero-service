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
- **json_name 全量显式化**：一方 proto 的所有字段必须显式带 `[json_name = "..."]`（2026-08 已覆盖全部 24 个文件，2580 个字段）。新增字段时同步补 tag，缺 tag 即视为契约不完整。
- **已知历史偏差（保留不动）**：`app/file/file.proto` `thumb_name → "ThumbName"`、`app/lalproxy/lalproxy.proto` `webUiVersion → "WebUiVersion"`、`facade/streamevent/streamevent.proto` `point_id → "PointId"` 三处 json_name 与 lowerCamelCase 约定不一致，但已固化在线上 descriptor/swagger 中属 wire 契约，**不得"顺手修正"**；确需修正须按破坏性变更走单独任务。
- **protoc ToJsonName 语义**：删除字段名中所有 `_` 并将其后字母大写（含数字前，如 `data_2nd` → `data2nd`）；以"重生成 Go 代码后 diff 为空"作为 json_name 与默认值一致的最终验证手段。

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
    Rule json.RawMessage `json:"rule"` // protojson 数据透传
}
```

**流程**：
```
写入适配: PlanRulePb → protojson.Marshal → json.RawMessage(Rule) → json.Marshal(CronJobExtra) → TaskConfig.Extra → 平铺到业务模型列
读取适配: 业务模型列 → 重建 CronJobExtra → json.Marshal → TaskConfig.Extra → json.Unmarshal → protojson.Unmarshal(Rule) → PlanRulePb
```

`json.Marshal` 写入 `json.RawMessage` 时原样输出字节，protojson 编码的 JSON 完整透传。这里的 `TaskConfig.Extra` 是运行时适配载体，不要求业务表保留同名 JSON 列；Trigger CronJob 和 ISP 均可把字段平铺到专属列后在读取时重建。

### Trigger Exact-Time Lists

- `CalcPlanTaskDateReq`、`CreatePlanTaskReq`、`CreateCronJobReq`、`UpdateCronJobReq`、`SubmitCronJobReq` 使用 `repeated string specified_times` / `excluded_times`，并显式标注 `json_name = "specifiedTimes"` / `"excludedTimes"`。
- 请求列表允许为空，每个列表至多 1000 项，每项长度必须为 19；格式和时区范围由 Trigger Schedule 编译器进一步校验。
- `CronJobPb` 回显同名字段但不重复请求校验。Proto 修改必须经 `app/trigger/gen.sh` 生成，业务转换不得直接编辑生成类型。
- gRPC JSON 使用 `specifiedTimes` / `excludedTimes`；两者是 repeated 完整配置字段，Update/Submit 中空数组或省略均表示空列表，不表示保留旧值。非空项由 Trigger 编译器按 `Asia/Shanghai` 解析并校验位于规范化闭区间。

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
- 服务间回调/事件请求只携带下游业务处理所需的关键字段，不把调用方内部运行状态（lease、`next_run`、调度规则等）复制进契约；回调契约是否需要 PGV validation 由契约所有方显式决定，不默认引入。
- 契约所有方明确允许覆盖时，字段号可直接顺延对齐（从 1 连续排列）并在任务文档中声明不保留兼容；Go 字段名由 proto 字段名派生，仅调整序号不影响 Go 调用方。

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
