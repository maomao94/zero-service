# Codec & Serializer

> **EXPERIMENTAL** — 此包尚未经过生产环境验证。

## CodecConn 接口

Codec 不再接收 `*Session`，改为最小接口 `CodecConn`，仅提供只读访问：

```go
// common/gnetx/codec.go:14-17
type CodecConn interface {
    ID() string
    Attribute(key any) any
}
```

## Codec 接口

```go
// common/gnetx/codec.go:30-33
type Codec interface {
    Decode(c gnet.Conn, sc CodecConn) (any, error)
    Encode(msg any, sc CodecConn) ([]byte, error)
}
```

### 线程契约

| 方法 | 执行上下文 | 可用操作 | 禁止 |
|------|-----------|---------|------|
| `Decode` | `OnTraffic`（event-loop） | `Peek`/`Discard`/`InboundBuffered` | `Read`/业务阻塞 |
| `Encode` | on-loop（`gc.Write`）或 off-loop（`AsyncWrite`） | 序列化 | 读 conn |

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
    Decode(raw []byte, sc CodecConn) (any, error)
    Encode(msg any, sc CodecConn) ([]byte, error)
}
```

序列化器只管 raw 字节 ↔ 消息结构转换。内置 Codec 在调用 `Serializer.Decode` 前已将帧数据 copy 成独立切片。

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
func (c *myCodec) Decode(conn gnet.Conn, sc CodecConn) (any, error) { ... }
func (c *myCodec) Encode(msg any, sc CodecConn) ([]byte, error) { ... }
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
