# ISP 协议接入指南

> ISP（Inspection Substation Protocol）= 区域型变电站远程智能巡视系统技术规范，`common/isp` 包提供协议编解码和常量定义。

## 何时阅读

- 新增 `ispagent` 业务 handler
- 对接上级巡检系统（ISP 协议服务端）
- 扩展 ISP 协议常量或消息类型

## 协议概述

传输层 TCP，帧格式：

```
0xEB90(2B BE) + SendSeq(8B LE) + RecvSeq(8B LE) + SessionSource(1B) + XMLLength(4B LE) + XML(UTF-8) + 0xEB90(2B BE)
```

- 0xEB90 大端，SendSeq/RecvSeq/XMLLength 小端
- `messageId = (Type << 16) | Command`
- Command=0 为上报类，Command≠0 为指令类
- XML 根元素可配置 `PatrolHost` / `PatrolDevice`

## 包结构

```
common/isp/
├── constants.go      # Type/Command/MessageID 常量 + 工具函数
├── message.go        # Message 结构 + gnetx 接口实现
├── xml.go            # XML 编解码（BuildXML/ParseXML）
├── serializer.go     # gnetx Codec 构造 + ISP Serializer
└── serializer_test.go
```

## 编解码

使用 `gnetx.LengthPrefixCodec`，带 `leadingBytes=0xEB90`、`trailingBytes=0xEB90`：

```go
codec := isp.NewCodec(rootName, maxFrameLength, debug)
```

- `stripBytes=2`：只剥前导，保留 21B 头给 Serializer
- `lengthOffset=19`、`lengthAdjust=2`：XMLLength 不含尾缀
- `debug=true`：启用 `gnetx.DebugSerializer`（debug 级别输出 hex）

## Message 模型

- `Identifiable` → `MessageID()` 供 Router 路由
- `Correlatable` → `TID()` = SendSeq 供请求-响应匹配
- `Response` → `ResponseTID()` = RecvSeq 供回包解回
- `SendSeq`/`RecvSeq` 对应协议帧 sendSerialNo/receiveSerialNo
- `RecvSeq` 为 ACK（上次收到的对端 SendSeq），出站时回执

## Item 约定

XML 中 `<Item attr="value"/>` 解析为 `map[string]string`，协议定义明确前保持动态。

## 注册响应

251-4 响应 Item 中的 `heart_beat_interval` 属性（秒）覆盖默认心跳间隔。

## 应答约定

所有 service→client 消息需回复 251-3 通用应答（Code 区分 `100/200/400/500`），handler 通过 `responseWithCode()` 构造。

## 常见错误

| 错误 | 说明 |
|------|------|
| Command=0 时输出 `<Command>0</Command>` | 应省略（`xmlMessage.Command` 使用 `omitempty`） |
| SendSeq/RecvSeq 字节序用混 | SendSeq/RecvSeq/XMLLength 一律小端 |
| 混淆 Type 共用 | Type=1 既是巡视设备状态上报也是机器人本体指令，由 Command 区分 |
