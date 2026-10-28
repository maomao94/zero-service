package dispatch

import (
	"bytes"

	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// decode 将 data 按 protojson 解析到 msg。
// 空 data 视为 {}，方便无参数方法调用；未知字段与类型错误会被拒绝。
func decode(data string, msg proto.Message) error {
	raw := bytes.TrimSpace([]byte(data))
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(raw, msg); err != nil {
		return tool.NewErrorByPbCode(extproto.Code__1_00_PARAM, "data 不是合法的 protojson: %v", err)
	}
	return nil
}

// Encode 将 proto 消息编码为 protojson，保持 int64 为字符串、枚举为名称。
func Encode(msg proto.Message) ([]byte, error) {
	return protojson.MarshalOptions{}.Marshal(msg)
}
