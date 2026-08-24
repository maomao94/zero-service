package broadcast

import (
	"context"
	"errors"
	"time"

	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
)

// Executor 业务执行器：处理一条已解码的广播请求并返回业务结果字节。
//
// 骨架约定：
//   - 返回 (result, nil)：成功，回 ack Success=true，ResponseBody=string(result)
//   - 返回 (nil, ErrSkipAck)：业务成功但**不回 ack**（非本节点业务对象，由持有节点回 ack）
//   - 返回 (nil, err)：失败，回 ack Success=false 且 ErrorKind=NormalizeErrorKind(err)
//
// 骨架不做业务反序列化：payload 是 BroadcastBody.Body 的字节（如 protojson 字符串、
// taskId 原文），解析由 executor 自行约定完成。
type Executor func(ctx context.Context, method string, payload []byte) ([]byte, error)

// ConsumeBroadcast 广播消费入口（满足 mqttx.ConsumeHandler 签名，可通过
// mqttx.Client.AddHandlerFunc(BroadcastTopicPattern(prefix), b.ConsumeBroadcast) 注册）。
//
// 流程：jsonx 反序列化 → 自身 ack 回环忽略 → executor 分发（method+payload）→
// 成功/失败回 ack（含 errorKind 归一）。
func (b *broadcaster) ConsumeBroadcast(ctx context.Context, payload []byte, topic string, topicTemplate string) error {
	body := &BroadcastBody{}
	if err := jsonx.Unmarshal(payload, body); err != nil {
		return err
	}

	// 防回环：忽略自身发布的消息（AckTopic 指向本实例的 ack 通道）
	if body.AckTopic == b.ackTopic() {
		logx.WithContext(ctx).Debugw("mqtt broadcast loopback ignored",
			logx.Field("tid", body.Tid),
			logx.Field("method", body.Method),
		)
		return nil
	}
	logx.WithContext(ctx).Infof("mqtt broadcast dispatch: method=%s tid=%s ackTopic=%s", body.Method, body.Tid, body.AckTopic)

	exec, ok := b.executor(body.Method)
	if !ok {
		// 未注册 method：回 ack 失败并携带 unknown 类别（调用方可直接感知不支持）
		b.publishAck(ctx, body.AckTopic, &BroadcastAckBody{
			Tid:       body.Tid,
			Method:    body.Method,
			Success:   false,
			Error:     "unknown method",
			ErrorKind: KindUnknown,
		})
		return nil
	}

	result, err := exec(ctx, body.Method, []byte(body.Body))
	if err != nil {
		if errors.Is(err, ErrSkipAck) {
			logx.WithContext(ctx).Debugw("mqtt broadcast skip ack",
				logx.Field("tid", body.Tid),
				logx.Field("method", body.Method),
			)
			return nil
		}
		logx.WithContext(ctx).Errorw("mqtt broadcast executor failed",
			logx.Field("tid", body.Tid),
			logx.Field("method", body.Method),
			logx.Field("error", err),
		)
		b.publishAck(ctx, body.AckTopic, &BroadcastAckBody{
			Tid:       body.Tid,
			Method:    body.Method,
			Success:   false,
			Error:     err.Error(),
			ErrorKind: NormalizeErrorKind(err),
		})
		return nil
	}

	// 成功：骨架填充关联字段与 Success，executor 返回的业务字节进 ResponseBody
	b.publishAck(ctx, body.AckTopic, &BroadcastAckBody{
		Tid:          body.Tid,
		Method:       body.Method,
		Success:      true,
		ResponseBody: string(result),
	})
	return nil
}

// publishAck 向请求方的 ack 通道回发结果（PublishWithTrace，10s 推送超时）。
// 无 Tid 或 AckTopic 为空时无法关联，直接跳过。
func (b *broadcaster) publishAck(ctx context.Context, ackTopic string, ack *BroadcastAckBody) {
	if ack == nil || ack.Tid == "" || ackTopic == "" {
		return
	}
	if b.client == nil {
		logx.WithContext(ctx).Error("mqtt broadcast client is nil")
		return
	}
	data, err := jsonx.Marshal(ack)
	if err != nil {
		logx.WithContext(ctx).Errorw("mqtt broadcast ack marshal failed", logx.Field("error", err))
		return
	}
	pushCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := b.client.PublishWithTrace(pushCtx, ackTopic, data); err != nil {
		logx.WithContext(pushCtx).Errorw("mqtt broadcast ack publish failed",
			logx.Field("tid", ack.Tid),
			logx.Field("method", ack.Method),
			logx.Field("ackTopic", ackTopic),
			logx.Field("error", err),
		)
	}
}

// executor 按 method 查找已注册执行器。
func (b *broadcaster) executor(method string) (Executor, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	fn, ok := b.executors[method]
	return fn, ok
}
