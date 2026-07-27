# 契约与生成规范

## 适用范围

修改 RPC/HTTP 方法、字段、校验、Swagger、事件入口或任何生成文件时读取。

## 契约源

- gRPC 以服务目录的 `.proto` 为源；HTTP/BFF 以 `.api` 为源。生成的 `*.pb.go`、`*_grpc.pb.go`、`*.pb.validate.go`、Server、Handler、Routes 和 Types 不是编辑入口。
- 使用目标服务根目录的 `gen.sh`。脚本可能包含 validate、grpc-gateway、OpenAPI 或不同 goctl 参数，不能用一条全仓通用命令替代。
- 外部共享 proto 位于 `third_party/`；修改前确认它是上游同步还是本项目所有的契约。
- `.proto` / `.api` 注释、validation、HTTP annotation 和实现行为要一起审查，不能只让代码编译。

依据：`app/trigger/trigger.proto`、`app/trigger/gen.sh`、`app/bridgegtw/bridgegtw.api`、`app/bridgegtw/gen.sh`、`aiapp/aisolo/aisolo.proto`、`aiapp/aisolo/gen.sh`。

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

## 验证

```bash
cd <service-directory>
./gen.sh
go test ./...
git diff --check
```

还要用 `rg` 搜索变更的字段名、RPC、事件名或枚举在仓库中的所有消费者。
