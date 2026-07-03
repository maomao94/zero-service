# implement.md: common/flowx

## 文件清单

| 文件 | 说明 |
|------|------|
| `common/flowx/options.go` | `FlowOptions` 配置结构体 + `FlowOption` + 便携选项函数 |
| `common/flowx/flowx.go` | `NewFlow` 构造函数 + `LoggingInterceptor` 实现 |
| `common/flowx/flowx_test.go` | 单元测试：默认行为、自定义选项、拦截器输出 |

## 执行顺序

1. `go get github.com/Azure/go-workflow` 添加依赖
2. 创建 `common/flowx/options.go`
3. 创建 `common/flowx/flowx.go`
4. 创建 `common/flowx/flowx_test.go`
5. `go build ./common/flowx/...` 验证编译
6. `go test ./common/flowx/...` 验证测试

## 验证命令

```bash
go build ./common/flowx/...
go test -v ./common/flowx/...
go vet ./common/flowx/...
```
