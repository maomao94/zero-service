# go-zero 服务约定

## 适用范围

修改 RPC/API 服务结构、Handler/Server、Logic、`ServiceContext` 或新增服务方法时读取。

## 分层契约

- Handler/Server 接收传输参数、调用 Logic 并返回结果；不要在生成入口堆业务编排、数据库事务或外部协议转换。
- Logic 拥有请求级业务流程、校验和多个依赖的协调；保持依赖显式来自 `ServiceContext` 或窄接口。
- `internal/svc/ServiceContext` 创建并保存服务共享的 client、store、scheduler、producer 等依赖，不保存请求级状态。
- Model/store/SDK 负责数据或外部系统边界；Logic 不应复制 SQL、topic、二进制帧或底层连接操作。
- 一个服务内沿用相邻 Logic 的构造与日志方式。不同服务的 `NewServiceContext` 可能返回单值或 `(*ServiceContext, error)`，不要为表面一致性改动无关服务。

依据：`app/trigger/internal/logic`、`app/trigger/internal/svc/servicecontext.go`、`app/trigger/internal/server`、`app/bridgegtw/internal/handler`、`aiapp/aisolo/internal/logic`。

## 开发顺序

1. 从 `.proto` 或 `.api` 确认输入、输出、校验和兼容性。
2. 运行服务自己的 `gen.sh`，不要从其他服务复制生成命令。
3. 在 Logic 中实现业务，在 `ServiceContext` 中装配新增共享依赖。
4. 把可复用且传输中立的能力留在现有 `common/` 包或窄接口后方。
5. 运行目标服务测试/编译并检查生成 diff。

## 依赖方向

- 传输层可以依赖 Logic；Logic 可以依赖服务私有 store/SDK 和公共包。
- `common/` 不得反向依赖具体服务的 `internal/`、生成 pb 或业务模型。
- 跨服务调用使用生成 client 或现有 facade，不直接导入另一个服务的 `internal/`。
- 多个入口共享业务时提取服务私有组件或公共领域接口，避免 Handler 互相调用。

## 反模式

- 手写或长期修改生成的 Server/Handler/Routes/Types。
- 把输入校验、事务、重试和外部调用全部塞进 Server 方法。
- 为单个 Logic 创建全局单例或把请求状态放进 `ServiceContext`。
- 从另一个服务复制配置和生成脚本而不核对本服务插件与输出目录。

## 验证

```bash
cd <service-directory>
./gen.sh
go test ./...
git diff --check
```

如果只改手写 Logic 且契约未变，不应制造生成文件 diff；至少运行目标服务或受影响包测试。
