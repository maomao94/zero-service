package broadcast

import (
	"context"
	"time"

	"zero-service/common/mqttx"

	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
)

// defaultAckReplyTTL ack 应答等待的默认 TTL（Broadcaster 与 AckReplyRouter 共用）。
const defaultAckReplyTTL = 10 * time.Second

// decodeAck 通用广播 ack 解码器（原两 app 的 ack 解码逻辑并集实现）。
// 协议规则：jsonx 反序列化为 BroadcastAckBody，Tid 为空则返回 mqttx.ErrEmptyReplyTid。
func decodeAck(ctx context.Context, payload []byte, topic string, topicTemplate string) (mqttx.ReplyMessage[*BroadcastAckBody], error) {
	ackBody := &BroadcastAckBody{}
	if err := jsonx.Unmarshal(payload, ackBody); err != nil {
		return mqttx.ReplyMessage[*BroadcastAckBody]{}, err
	}
	if ackBody.Tid == "" {
		return mqttx.ReplyMessage[*BroadcastAckBody]{}, mqttx.ErrEmptyReplyTid
	}
	logx.WithContext(ctx).Debugw("mqtt broadcast ack received",
		logx.Field("tid", ackBody.Tid),
		logx.Field("method", ackBody.Method),
		logx.Field("topic", topic),
		logx.Field("topic_template", topicTemplate),
		logx.Field("success", ackBody.Success),
		logx.Field("error_kind", ackBody.ErrorKind),
	)
	return mqttx.ReplyMessage[*BroadcastAckBody]{
		Tid:   ackBody.Tid,
		Value: ackBody,
	}, nil
}

// NewAckReplyRouter 构造广播 ack 应答路由（供业务在创建 mqttx.Client 时经
// mqttx.WithReplyRouter(BroadcastAckTopic(prefix, instanceID), router) 注册）。
//   - ttl <= 0 时使用默认 10s（与 Broadcaster 的默认 TTL 一致）
//   - name 为空时使用默认路由名
func NewAckReplyRouter(ttl time.Duration, name string) *mqttx.ReplyRouter[*BroadcastAckBody] {
	if ttl <= 0 {
		ttl = defaultAckReplyTTL
	}
	if name == "" {
		name = "mqttx-broadcast-reply"
	}
	return mqttx.NewReplyRouter[*BroadcastAckBody](
		mqttx.ReplyDecoderFunc[*BroadcastAckBody](decodeAck),
		mqttx.WithReplyRouterName(name),
		mqttx.WithReplyRouterTTL(ttl),
	)
}
