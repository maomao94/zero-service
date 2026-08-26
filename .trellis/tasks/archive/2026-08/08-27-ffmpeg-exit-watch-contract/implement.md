# 执行计划：FFmpeg 退出与 stdout 契约

## 步骤

1. 先更新 `common/ffmpegx` 测试：重复 ID、ExitResult、cancel/timeout 下 WaitErr 与 ContextErr、逐行 stdout 同步回调。
2. 实现 `ErrProcessExists`、`ExitResult`、`WithStdoutHandler` 和 `WatchOutput`，删除 progress 专属公共类型/API。
3. 更新 `PullRegistry`：逐行解析 progress，传递 ExitResult，支持绝对 deadline context。
4. 更新 `RelayState`、`DistributedRelay` 与退出/补拉决策，确保 deadline 到期不启动或补拉。
5. 在 proto 增加 `max_duration_seconds`，运行 `app/oryxserver/gen.sh`，更新 logic 与所有调用方。
6. 补 relay 测试：0/正数时长、Reconcile 剩余时间、deadline 到期、流异常和手动 Stop。
7. 更新 Trellis 并发与 Oryx 规范，执行完整验证。

## 验证

```bash
go test ./common/ffmpegx/... ./app/oryxserver/internal/relay/... ./app/oryxserver/internal/logic/...
go test -race -count=10 ./common/ffmpegx/... ./app/oryxserver/internal/relay/...
go build ./...
go vet ./common/ffmpegx/... ./app/oryxserver/...
go test ./...
git diff --check
```

## 风险与回滚

- `WithProgressHandler`/`Progress` 是公共 API 删除，需要全仓搜索并迁移所有调用方。
- 生成脚本可能产生大范围格式差异，必须核对生成 diff 只包含新字段。
- rollback 时恢复旧 handler API；新增 proto 字段可保留为未使用字段，不破坏 wire 兼容。
