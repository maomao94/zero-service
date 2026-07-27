# 质量规范

## 适用范围

制定实施计划、补测试、执行生成脚本、审查 diff 和交付结果时读取。

## 风险决定验证范围

| 变更 | 最低验证 |
| --- | --- |
| 文档/Spec | 链接或路径检查、占位检查、`git diff --check` |
| 纯函数/转换 | 目标包单元测试，覆盖边界和错误输入 |
| Logic/Handler | 目标服务编译或测试，验证依赖和契约连接 |
| `.proto` / `.api` | 对应 `gen.sh`、生成 diff、目标服务测试 |
| 数据访问/迁移 | 目标 model/store 测试，检查事务、条件更新、空值和方言差异 |
| 并发/连接/调度 | 状态与失败路径测试，必要时 `go test -race` 和重复运行 |
| 公共 API/跨服务 | 所有直接调用方测试，再评估 `go test ./...` / `go vet ./...` |

先运行最小相关集合，扩大范围要有依赖或风险依据。外部数据库、Redis、MQTT、设备或凭据不可用时，说明未验证项，不伪造成功。

## 必查内容

- 契约源、生成产物、实现和文档是否一致。
- 成功、业务失败、超时/取消、重复调用、空值和并发竞争是否有明确行为。
- 持久化更新是否只拥有自己的字段，并用事务、唯一约束、版本或 CAS 保护竞争路径。
- 日志和错误是否保留 trace/业务标识但不泄露敏感数据。
- 新依赖是否必要；只有实际修改依赖时运行 `go mod tidy` 并审查 `go.mod`、`go.sum`。
- Git diff 是否只包含任务范围，生成代码是否来自生成脚本。

## 测试风格

- 优先测试可观察契约，不依赖未导出实现细节或固定 goroutine 调度顺序。
- 时间、重试和异步回执使用有上限的轮询或可控时钟，不使用无界等待和脆弱的固定长睡眠。
- 数据库测试显式断言 `RowsAffected`、错误、状态与关键字段，不只断言查询成功。
- 并发测试必须能失败于错误的所有权/CAS，而不是只通过 `go test` 证明“没有立刻报错”。
- 生成代码通常不单独测试；测试源契约、Logic、公共包和数据转换。

依据：`common/crontask/*_test.go`、`common/gormx/*_test.go`、`common/djisdk/protocol_drc_test.go`、`app/ispagent/internal/handler/*_test.go`。

## 反模式

- 用全仓测试替代对真实风险的定向断言。
- 只覆盖成功路径，忽略取消、重复回调、过期 lease、终止状态或部分失败。
- 为让测试通过而扩大生产代码改动或修改不相关生成文件。
- 测试未运行却用“应该通过”表述。

## 验证

```bash
gofmt -w <changed-go-files>
go test ./path/to/package
go test -race ./path/to/concurrent/package
go vet ./path/to/package
go test ./...
git diff --check
git status --short
```

只执行与本次风险相符的命令，并在交付时列出实际结果。
