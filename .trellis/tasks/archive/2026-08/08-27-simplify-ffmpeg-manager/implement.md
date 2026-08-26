# 执行计划：简化 FFmpeg Manager

## 步骤

1. 先修改 `common/ffmpegx/process_test.go`，删除阻塞 builder/同 ID 并发操作测试，增加同步 exit/progress callback、context cancel/timeout、顺序 replacement 和不同 ID 非阻塞测试。
2. 简化 `common/ffmpegx/process.go`：删除状态机、AfterFunc、参数扫描和启动 channel；Manager 用 `context.WithCancel(ctx)` 派生 builder/command 共用的 process context，并改用 `RWMutex` 与最小 process 字段。
3. 更新 progress option 注释和 relay registry 测试，确认 relay 命令显式配置 stdout progress。
4. 更新并发与 Oryx 规范，删除自动参数检测和并发同 ID 承诺。
5. 运行 gofmt、目标测试、race、全仓 build/test、目标 vet 和 diff 检查。

## 验证命令

```bash
go test ./common/ffmpegx/... ./app/oryxserver/internal/relay/...
go test -race -count=10 ./common/ffmpegx/... ./app/oryxserver/internal/relay/...
go build ./...
go vet ./common/ffmpegx/... ./app/oryxserver/...
go test ./...
git diff --check
```

## 风险与回滚

- 同 ID 并发 Start/Stop 不再是支持契约；调用方必须串行化同 ID 操作。
- 若 relay 测试暴露并发调用，优先在业务 registry 的同 target 边界串行化，不恢复通用 Manager 状态机。
- 可回滚到提交 `0b3ad277` 的 Manager 实现。
