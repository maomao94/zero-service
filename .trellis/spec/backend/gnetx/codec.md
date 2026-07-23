# Codec & Serializer

> **EXPERIMENTAL** — 此包尚未经过生产环境验证。

## Codec 接口

```go
// common/gnetx/codec.go
type Codec interface {
    Decode(c gnet.Conn, conn Conn) (any, error)
    Encode(ctx context.Context, msg any, conn Conn) ([]byte, error)
}
```

`CodecConn` 已删除；Codec 直接接收公共 `Conn` 接口，用于读取 session 元数据、地址、attributes，并通过 `NextSendSeq()` 获取连接级发送序号。

### 线程契约

| 方法 | 执行上下文 | 可用操作 | 禁止 |
|------|-----------|---------|------|
| `Decode` | `OnTraffic`（event-loop） | `gnet.Conn.Peek`/`Discard`/`InboundBuffered`、`Conn` 元数据 | `Read`/业务阻塞 |
| `Encode` | on-loop（`gc.Write`）或 off-loop（`AsyncWrite`） | `context.Context`、`Conn` 元数据、`NextSendSeq()`、序列化 | 读 `gnet.Conn` / 直接写连接 |

`gnet.Conn` 只出现在 `Decode`，因为它暴露 event-loop-only buffer API。`Encode` 不接收 `gnet.Conn`，避免 off-loop 发送路径误用 `Peek`/`Discard`/`Write`。

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
// common/gnetx/codec.go:38-41
type Serializer interface {
    Decode(raw []byte) (any, error)
    Encode(msg any) ([]byte, error)
}
```

序列化器只管 raw 字节 ↔ 消息结构转换，不接收 session/connection。协议头、序号、ack、CRC、TID 映射等上下文相关逻辑必须留在 Codec。内置 Codec 在调用 `Serializer.Decode` 前已将帧数据 copy 成独立切片。

## 内置 Codec

### LengthPrefixCodec — 长度前缀分帧

```go
codec := gnetx.NewLengthPrefixCodec(4, binary.BigEndian, gnetx.JSONSerializer{},
    gnetx.WithMaxFrameSize(1 << 20),
)
```

`common/gnetx/codec_lengthprefix.go:64-86`（构造）、`:93-126`（Decode）、`:138-158`（Encode）

**Decode 安全校验顺序**（`:101-112`）：
1. Payload 长度 < 0（`lengthAdjust` 或 uint64→int 溢出）→ `ErrFrameTooLarge`
2. Frame 总长溢出 → `ErrFrameTooLarge`
3. 超过 `maxFrameSize` → `ErrFrameTooLarge`

**Encode 防截断**：payload 超长度字段容量时返回 error，不静默截断（`:148-151`）。

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
func (c *myCodec) Decode(gconn gnet.Conn, conn Conn) (any, error) { ... }
func (c *myCodec) Encode(ctx context.Context, msg any, conn Conn) ([]byte, error) { ... }
```

`common/gnetx/codec.go:60-73`

半包错误转换：用 `mapShortBuffer(err)` 把 `io.ErrShortBuffer` 映射为 `ErrIncompletePacket`。
`common/gnetx/codec.go:77-85`

## MaxFrameLength 注入

`NewServer`/`NewClient` 把必填的 `MaxFrameLength` 注入到内置 Codec：

```go
// common/gnetx/codec.go:48-57
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

## DebugSerializer hex 日志格式

`DebugSerializer` 包装任意 `Serializer`，在 debug 级别打印入站/出站 payload。默认格式必须保持为 `hex.EncodeToString` 等价的小写紧凑 hex，避免破坏既有日志检索：

```go
ser := gnetx.DebugSerializer(inner)
// [gnetx] recv 4 bytes hex=680e00ff
```

协议现场排查需要按字节查看时，用 `WithDebugHexFormat` 显式 opt in：

```go
ser := gnetx.DebugSerializer(inner, gnetx.WithDebugHexFormat(gnetx.HexUpperSpace))
// [gnetx] recv 4 bytes hex=68 0E 00 FF
```

### Signatures

```go
type DebugHexFormat int

const (
    HexLowerCompact DebugHexFormat = iota
    HexUpperCompact
    HexUpperSpace
)

type DebugSerializerOption func(*debugSerializerOptions)

func WithDebugHexFormat(format DebugHexFormat) DebugSerializerOption
func DebugSerializer(inner Serializer, opts ...DebugSerializerOption) Serializer
```

底层通用字节格式化放在 `common/tool`：

```go
func tool.HexBytes(raw []byte, format tool.HexBytesFormat) string
```

### Contracts

- `DebugSerializer(inner)` 默认输出小写紧凑 hex，等价于 `tool.HexBytes(raw, tool.HexLowerCompact)`。
- `HexUpperCompact` 输出大写紧凑 hex，例如 `680E00FF`。
- `HexUpperSpace` 输出大写且按字节空格分隔，例如 `68 0E 00 FF`；实现可直接使用 `fmt.Sprintf("% X", raw)`。
- 该配置只影响日志文本，不得改变 `Serializer.Decode` / `Serializer.Encode` 的入参、返回值、错误传播或 codec 分帧行为。

### Tests Required

- `common/tool` 单测覆盖 lower compact、upper compact、upper space 和未知格式默认值。
- `common/gnetx` 单测覆盖 `DebugSerializer` 接收 `WithDebugHexFormat`，并确认 `formatDebugHex` 映射到期望格式。
- 修改该能力后至少运行 `go test -race -count=1 ./common/gnetx/... ./common/tool/...`；如因既有网络时序测试失败，需补跑聚焦测试并说明失败与本改动无关。

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
| 在 Encode 中尝试访问 `gnet.Conn` | Encode 可能 off-loop 执行，不能读写 gnet buffer |
| 把协议头逻辑写进 Serializer | Serializer 是 body-only，协议上下文应由 Codec 处理 |
| 直接把 `DebugSerializer` 默认改成 `% X` | 会破坏既有小写紧凑 hex 日志检索；协议排查格式必须 opt in |

## Codec/Serializer Interface Contract

### 1. Scope / Trigger

- Trigger: 修改 gnetx 编解码公开接口或新增协议 Codec/Serializer。

### 2. Signatures

```go
type Codec interface {
    Decode(c gnet.Conn, conn Conn) (any, error)
    Encode(ctx context.Context, msg any, conn Conn) ([]byte, error)
}

type Serializer interface {
    Decode(raw []byte) (any, error)
    Encode(msg any) ([]byte, error)
}
```

### 3. Contracts

- `Decode` may use `gnet.Conn` buffer APIs only inside OnTraffic.
- `Encode` must only produce bytes; framework owns `Write`/`AsyncWrite`.
- `ctx` may carry protocol request context for reply encoding (injected by framework via `PacketContextProvider`).
- `conn.NextSendSeq()` is the framework-provided connection sequence allocator.

### 4. Validation & Error Matrix

| Condition | Error/Behavior |
| --- | --- |
| Incomplete frame | return `ErrIncompletePacket` and consume no bytes |
| Frame exceeds max | return `ErrFrameTooLarge` |
| Serializer cannot encode type | return codec/serializer error; caller does not write |

### 5. Good/Base/Bad Cases

- Good: Codec parses header, message implements PacketContextProvider, dispatch injects pc into ctx. Encode reads pc from ctx for reply header. Serializer only maps body bytes to structs.
- Base: Built-in LengthPrefix/Delimiter/Fixed codecs ignore ctx and use body-only serializers.
- Bad: Serializer calls `conn.NextSendSeq()` or parses ack/CRC.

### 6. Tests Required

- Codec tests cover full frame, half frame, frame-too-large, encode length overflow.
- 8 字节长度字段必须覆盖 `uint64 > MaxInt` 且带正 `lengthAdjust` 的 Decode 溢出用例。
- Serializer tests cover body encode/decode only.
- Runtime tests verify ctx is passed to `Encode` and sequence start is honored.

### 7. Wrong vs Correct

#### Wrong

```go
func (c *myCodec) Encode(msg any, gconn gnet.Conn) ([]byte, error) {
    _, _ = gconn.Peek(1)
    return nil, nil
}
```

#### Correct

```go
func (c *myCodec) Encode(ctx context.Context, msg any, conn Conn) ([]byte, error) {
    seq := conn.NextSendSeq()
    _ = seq
    return c.encodeFrame(ctx, msg)
}
```
