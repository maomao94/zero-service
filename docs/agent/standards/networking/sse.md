# SSE Writer

修改 `common/ssex` 或 SSE handler 的帧写入、缓冲刷新和结束语义时读取。

### Writer API

```go
w, _ := ssex.NewWriter(responseWriter)
w.WriteData("hello")           // data: hello\n\n
w.WriteEvent("custom", "msg")  // event: custom\ndata: msg\n\n
w.WriteJSON(struct{...})       // data: {"...\n\n
w.WriteDone()                  // data: [DONE]\n\n  (OpenAI 兼容)
w.WriteKeepAlive()             // : keepalive\n\n
w.WriteComment("debug info")   // : debug info\n\n
```

- `Writer` 实现 `io.Writer` 接口，支持流式 A2UI 输出。
- 行缓冲: `Write()` 累积字节直到 `\n`，按行发送 `data: {line}\n\n`。
- 线程安全: 所有公开方法使用 `sync.Mutex` 串行化。
- 向后兼容别名: `LineWriter` = `Writer`、`NewLineWriter` = `NewWriter`。
- `ResponseWriter()` 暴露底层 writer。

依据：`common/ssex/writer.go`。

### 反模式 (ssex)

- 不设置 Content-Type header（SSE 需要 `text/event-stream`）。
- 使用 `Write()` 后不调用 `BufferFlush()` 发送缓冲区残余。
- 在 SSE handler 中调用 `WriteDone()` 后继续写入。

验证行缓冲、所有写方法、并发安全和 OpenAI `[DONE]` 格式。
