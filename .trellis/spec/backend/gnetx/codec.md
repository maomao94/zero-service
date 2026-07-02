# Codec & Serializer

> **EXPERIMENTAL** — 此包尚未经过生产环境验证。

## Codec 接口

Codec 是单接口，同时承载分帧与序列化（对齐 gnet v1 ICodec 的简洁形态）：

```go
// common/gnetx/codec.go:23-26
type Codec interface {
    Decode(c gnet.Conn, sess *Session) (any, error)
    Encode(msg any, sess *Session) ([]byte, error)
}
```

### 线程契约

| 方法 | 执行上下文 | 可用操作 | 禁止 |
|------|-----------|---------|------|
| `Decode` | `OnTraffic`（event-loop） | `Peek`/`Discard`/`InboundBuffered` | `Read`/业务阻塞 |
| `Encode` | on-loop（`c.Write`）或 off-loop（`AsyncWrite`） | 序列化 | 读 conn |

### 半包处理

`Decode` 在缓冲区不足以凑成一帧时**必须**返回 `ErrIncompletePacket`，且不消费任何字节。

内置实现的两种 Peek 模式：
- **两段 Peek**（`LengthPrefixCodec`）：先 Peek 帧头得长度 → 校验 → 再 Peek 整帧 → Discard。半包时不 Discard。
- **全缓冲 Peek**（`DelimiterCodec`）：Peek(全部缓冲) → `bytes.Index` 找分隔符 → 找到则 Discard。

参考：
- `common/gnetx/codec_lengthprefix.go:93-126`
- `common/gnetx/codec_delimiter.go:66-100`
- `common/gnetx/codec_fixed.go:28-40`

## Serializer 接口

```go
// common/gnetx/codec.go:31-34
type Serializer interface {
    Decode(raw []byte, sess *Session) (any, error)
    Encode(msg any, sess *Session) ([]byte, error)
}
```

Serializer 只负责 raw 字节 ↔ 消息结构转换，与分帧层解耦。内置 Codec 在调用 `Serializer.Decode` 前已将帧数据 copy 成独立切片，因此 Serializer 可安全持有 raw 字节。

## 内置 Codec

### LengthPrefixCodec — 长度前缀分帧

最常用的二进制协议分帧方式。帧格式：`[lengthOffset 字节前缀][lengthBytes 字节长度字段][payload]`。

```go
codec := gnetx.NewLengthPrefixCodec(4, binary.BigEndian, gnetx.JSONSerializer{},
    gnetx.WithMaxFrameSize(1 << 20),
)
```

`common/gnetx/codec_lengthprefix.go:93-126`（Decode）、`:138-158`（Encode）

**Decode 安全校验顺序**：
1. Payload 长度 < 0（`lengthAdjust` 或 uint64→int 溢出）→ `ErrFrameTooLarge`
2. Frame 总长溢出 → `ErrFrameTooLarge`
3. 超过 `maxFrameSize` → `ErrFrameTooLarge`

**Encode 防截断**：payload 超长度字段容量时返回 error，不静默截断。

支持的参数（`LengthPrefixOption`）：
- `WithLengthOffset(offset)` — 长度字段前的跳过字节
- `WithLengthAdjust(adjust)` — 长度字段值与 payload 实际长度的差值调整
- `WithMaxFrameSize(max)` — 单帧上限

### DelimiterCodec — 分隔符分帧

```go
codec := gnetx.NewDelimiterCodec([]byte{0xEB, 0x90}, mySerializer)
```

`common/gnetx/codec_delimiter.go:66-100`（Decode）

⚠️ 大帧且多次到达时有 O(n²) 重复扫描，大二进制帧推荐用 LengthPrefix。

### FixedLengthCodec — 定长分帧

```go
codec := gnetx.NewFixedLengthCodec(16, mySerializer)
```

`common/gnetx/codec_fixed.go:28-40`（Decode）、`:45-47`（Encode）

## 自定义 Codec

```go
// NewFuncCodec：闭包构造
codec := gnetx.NewFuncCodec(myDecode, myEncode)

// 或直接实现 Codec 接口
type myCodec struct{}
func (c *myCodec) Decode(conn gnet.Conn, sess *Session) (any, error) { ... }
func (c *myCodec) Encode(msg any, sess *Session) ([]byte, error) { ... }
```

`common/gnetx/codec.go:52-66`

半包错误转换：用 `mapShortBuffer(err)` 把 `io.ErrShortBuffer` 映射为 `ErrIncompletePacket`。
`common/gnetx/codec.go:70-78`

## MaxFrameLength 注入

`NewServer`/`NewClient` 把必填的 `MaxFrameLength` 注入到内置 Codec：

```go
// common/gnetx/codec.go:41-50
type frameLimiter interface {
    applyMaxFrameSizeIfUnset(max int)
}
```

`LengthPrefixCodec` 和 `DelimiterCodec` 实现此接口；自定义 Codec 若不实现，需自行在 Decode 里限制帧长。

## 内置 Serializer

| Serializer | 用途 | Source |
|-----------|------|--------|
| `RawSerializer` | 透传 `[]byte`，原型/简单场景 | `common/gnetx/serializer.go:13-33` |
| `JSONSerializer` | JSON 序列化，返回 `map[string]any` | `common/gnetx/serializer.go:37-53` |

生产级二进制协议应自定义 Serializer 直接解析成 typed struct。

## 构造校验

所有内置 Codec 构造器在参数非法时 **panic**（early-fail 设计）：
- `lengthBytes` 非 1/2/4/8
- 空分隔符
- nil serializer

测试：`common/gnetx/codec_test.go:179-196`

## 常见错误

| 错误 | 说明 |
|------|------|
| Decode 用 `conn.Read` 而非 `Peek`/`Discard` | Read 消费字节，出错后无法重试 |
| 半包时 Discard 部分数据 | 下次可读事件拿不到完整帧头，协议错序 |
| Encode 不校验 payload 大小就 `putUintN` | 截断导致对端错帧（`uint16(70000)=4464`） |
| 自定义 Codec 未实现 `frameLimiter` 且无帧长校验 | `MaxFrameLength` 安全上限不生效 |
