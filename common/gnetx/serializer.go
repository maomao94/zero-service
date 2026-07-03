package gnetx

import "encoding/json"

// Serializer 契约备注：内置 Codec（LengthPrefix/Delimiter/Fixed）在调用 Serializer.Decode
// 前已把帧数据 copy 成独立切片，因此 Serializer 收到的 raw 可安全持有/返回给业务。
// 若你用 NewFuncCodec 自定义分帧且直接把 conn.Peek 的切片传给 Serializer，则需自行 copy。

// RawSerializer 是透传序列化器：把原始帧字节原样当作消息（[]byte）返回。
// 适用于想自己拿原始字节做 type-switch 解析、或协议本身就是纯字节流的场景。
// 主要用于原型/简单场景；生产级二进制协议建议自定义 Serializer 直接解析成 typed struct，
// 避免 []byte 中转。
type RawSerializer struct{}

// Decode 返回原始帧字节的独立副本。
// 再拷一次是防御性设计：即便某些自定义 Codec 传入的是可复用 buffer 切片，
// RawSerializer 返回给业务的 []byte 也始终安全可持有。内置 Codec 已 copy 时这层拷贝是冗余的，
// 但 RawSerializer 面向原型场景，可读性/安全优先。
func (RawSerializer) Decode(raw []byte, _ CodecConn) (any, error) {
	out := make([]byte, len(raw))
	copy(out, raw)
	return out, nil
}

// Encode 把消息序列化为字节。仅接受 []byte，其他类型返回 errRawSerializerType。
func (RawSerializer) Encode(msg any, _ CodecConn) ([]byte, error) {
	if b, ok := msg.([]byte); ok {
		out := make([]byte, len(b))
		copy(out, b)
		return out, nil
	}
	return nil, errRawSerializerType
}

// JSONSerializer 用 encoding/json 做 payload 序列化，适合快速原型和简单结构化协议。
// 生产级二进制协议建议自定义 Serializer。
type JSONSerializer struct{}

// Decode 把原始帧字节 JSON 反序列化为 map[string]any 返回。
// JSONSerializer 无法感知目标 Go 类型，故统一返回 map[string]any，由业务在 handler 里映射；
// 如需类型化（反序列化成具体 struct），请自定义 Serializer 用 json.Unmarshal 到目标类型。
func (JSONSerializer) Decode(raw []byte, _ CodecConn) (any, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// Encode 把消息 JSON 序列化为字节。
func (JSONSerializer) Encode(msg any, _ CodecConn) ([]byte, error) {
	return json.Marshal(msg)
}
