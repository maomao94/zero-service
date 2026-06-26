package mqttx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tool "zero-service/common/tool"
	tracex "zero-service/common/trace"

	"github.com/duke-git/lancet/v2/random"
	"github.com/duke-git/lancet/v2/slice"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// OpenTelemetry 属性 key
const (
	attrTopicTemplate = "mqtt.topic_template"
	attrClientID      = "mqtt.client_id"
	attrMsgID         = "mqtt.message_id"
	attrQoS           = "mqtt.qos"
	attrAction        = "mqtt.action"
)

// Client is the interface for mqttx operations.
// Callers should depend on this interface for publish, handler registration, and lifecycle management.
// RequestReply is a package-level generic function because Go does not support generic methods.
type Client interface {
	AddHandler(topicTemplate string, handler ConsumeHandler) error
	AddHandlerFunc(topicTemplate string, fn func(context.Context, []byte, string, string) error) error
	Publish(ctx context.Context, topic string, payload []byte) error
	PublishWithTrace(ctx context.Context, topic string, payload []byte) (string, error)
	Close()
	GetClientID() string
}

type replyHandlerGetter interface {
	getReplyHandler(topicTemplate string) ConsumeHandler
}

// mqttClient MQTT 客户端
// 提供 MQTT 连接、订阅、发布等功能
type mqttClient struct {
	client     mqtt.Client
	cfg        MqttConfig
	handlerMgr *handlerManager
	dispatcher *messageDispatcher
	subscribed map[string]struct{}
	onReady    func(Client)
	ready      atomic.Bool
	qos        byte
	mu         sync.RWMutex
	tracer     oteltrace.Tracer
	metrics    *stat.Metrics
}

// MustNewClient 创建 MQTT 客户端，连接失败时 panic
// 推荐在应用启动时使用，会自动注册关闭监听
func MustNewClient(cfg MqttConfig, opts ...ClientOption) Client {
	cli, err := NewClient(cfg, opts...)
	logx.Must(err)
	proc.AddShutdownListener(func() {
		cli.Close()
	})
	return cli
}

// NewClient 创建 MQTT 客户端
// 会自动连接 MQTT 服务器，重连由底层库处理
func NewClient(cfg MqttConfig, opts ...ClientOption) (Client, error) {
	if len(cfg.Broker) == 0 {
		return nil, errors.New("[mqtt] no broker addresses provided")
	}

	// 未指定 ClientID 时自动生成
	if len(cfg.ClientID) == 0 {
		uid, err := random.UUIdV4()
		if err != nil {
			return nil, fmt.Errorf("[mqtt] generate clientID failed: %w", err)
		}
		cfg.ClientID = strings.ReplaceAll(uid, "-", "")
	}

	adjustConfig(&cfg)

	o := &ClientOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}

	c := &mqttClient{
		cfg:        cfg,
		handlerMgr: newHandlerManager(),
		subscribed: make(map[string]struct{}),
		qos:        cfg.Qos,
		onReady:    o.onReady,
		tracer:     otel.Tracer(trace.TraceName),
		metrics:    stat.NewMetrics(fmt.Sprintf("mqtt-%s", cfg.ClientID)),
	}
	c.dispatcher = newMessageDispatcher(c.handlerMgr, c.metrics)

	for _, reg := range o.replyRouters {
		c.handlerMgr.addReplyHandler(reg.topicTemplate, reg.handler)
	}

	c.client = c.createMqttClient()
	if err := c.connect(); err != nil {
		return nil, err
	}

	return c, nil
}

// adjustConfig 修正配置默认值
func adjustConfig(cfg *MqttConfig) {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30000
	}
	if cfg.KeepAlive <= 0 {
		cfg.KeepAlive = 60000
	}
	if cfg.Qos > 2 {
		cfg.Qos = 1
	}
}

// createMqttClient 创建底层 MQTT 客户端
func (c *mqttClient) createMqttClient() mqtt.Client {
	optsMqtt := mqtt.NewClientOptions()
	for _, broker := range c.cfg.Broker {
		optsMqtt.AddBroker(broker)
	}

	optsMqtt.SetClientID(c.cfg.ClientID).
		SetUsername(c.cfg.Username).
		SetPassword(c.cfg.Password).
		SetAutoReconnect(true).
		SetConnectTimeout(time.Duration(c.cfg.Timeout) * time.Millisecond).
		SetKeepAlive(time.Duration(c.cfg.KeepAlive) * time.Millisecond)

	optsMqtt.SetDefaultPublishHandler(c.defaultHandler)
	optsMqtt.OnConnect = c.onConnect
	optsMqtt.OnConnectionLost = c.onConnectionLost

	return mqtt.NewClient(optsMqtt)
}

func (c *mqttClient) connect() error {
	token := c.client.Connect()

	if !token.WaitTimeout(time.Duration(c.cfg.Timeout) * time.Millisecond) {
		return fmt.Errorf("[mqtt] connect timeout after %dms", c.cfg.Timeout)
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("[mqtt] connect failed: %w", err)
	}

	return nil
}

// onConnect 连接成功回调
func (c *mqttClient) onConnect(_ mqtt.Client) {
	logx.Infof("[mqtt] connected client=%s", c.cfg.ClientID)

	// 首次连接执行 onReady 回调
	if c.ready.CompareAndSwap(false, true) && c.onReady != nil {
		c.onReady(c)
	}

	// 恢复订阅
	if result, err := c.restoreSubscriptions(); err != nil {
		logx.Errorf("[mqtt] restore subscriptions failed err=%v", err)
	} else {
		logx.Infof("[mqtt] restore subscriptions done subscribed=%d skipped=%d", result.subscribed, result.skipped)
	}
}

// onConnectionLost 连接丢失回调
func (c *mqttClient) onConnectionLost(_ mqtt.Client, err error) {
	logx.Errorf("[mqtt] connection lost err=%v", err)
	c.mu.Lock()
	c.subscribed = make(map[string]struct{}) // 重连后需要重新订阅
	c.mu.Unlock()
}

// defaultHandler 默认消息处理器（无自定义处理器时）
func (c *mqttClient) defaultHandler(_ mqtt.Client, msg mqtt.Message) {
	logx.Errorf("[mqtt] no handler topic=%s", msg.Topic())
}

// AddHandler 为订阅主题模板添加消息处理器。
// 如果已连接且 topic 尚未订阅，会自动订阅。
// reply router 请使用 WithReplyRouter，不要把 ReplyRouter 传入此方法。
func (c *mqttClient) AddHandler(topicTemplate string, handler ConsumeHandler) error {
	if handler == nil {
		return errors.New("[mqtt] handler cannot be nil")
	}

	c.handlerMgr.addHandler(topicTemplate, handler)

	c.mu.Lock()
	needSubscribe := !c.isSubscribed(topicTemplate)
	c.mu.Unlock()

	if needSubscribe && c.client.IsConnected() {
		_, err := c.subscribe(topicTemplate)
		return err
	}
	return nil
}

// AddHandlerFunc 快捷方法：用函数作为消息处理器
func (c *mqttClient) AddHandlerFunc(topicTemplate string, fn func(context.Context, []byte, string, string) error) error {
	return c.AddHandler(topicTemplate, ConsumeHandlerFunc(fn))
}

// subscribe 内部订阅实现
func (c *mqttClient) subscribe(topicTemplate string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isSubscribed(topicTemplate) {
		return false, nil
	}

	token := c.client.Subscribe(topicTemplate, c.qos, c.messageHandler(topicTemplate))
	timeout := time.Duration(c.cfg.Timeout) * time.Millisecond

	if !token.WaitTimeout(timeout) {
		return false, fmt.Errorf("[mqtt] subscribe to %s timeout after %v", topicTemplate, timeout)
	}
	if err := token.Error(); err != nil {
		return false, err
	}

	c.subscribed[topicTemplate] = struct{}{}
	logx.Infof("[mqtt] subscribed topic=%s", topicTemplate)
	return true, nil
}

// isSubscribed 检查是否已订阅
func (c *mqttClient) isSubscribed(topicTemplate string) bool {
	_, exists := c.subscribed[topicTemplate]
	return exists
}

// restoreSubscriptions 恢复所有订阅（连接重连后）
func (c *mqttClient) restoreSubscriptions() (restoreSubscriptionsResult, error) {
	topicTemplates := c.handlerMgr.getAllTopicTemplates()
	topicTemplates = append(topicTemplates, c.cfg.SubscribeTopics...)
	topicTemplates = uniqueTopics(topicTemplates)

	var lastErr error
	result := restoreSubscriptionsResult{}
	for _, topicTemplate := range topicTemplates {
		subscribed, err := c.subscribe(topicTemplate)
		if err != nil {
			logx.Errorf("[mqtt] subscribe failed topic=%s err=%v", topicTemplate, err)
			lastErr = err
			continue
		}
		if subscribed {
			result.subscribed++
		} else {
			result.skipped++
		}
	}
	return result, lastErr
}

type restoreSubscriptionsResult struct {
	subscribed int
	skipped    int
}

func uniqueTopics(topics []string) []string {
	return slice.Unique(topics)
}

// messageHandler 消息处理包装器
// topicTemplate 参数为触发当前回调的订阅模板（可能含通配符 +/#），dispatcher 使用该模板精确路由。
func (c *mqttClient) messageHandler(topicTemplate string) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		c.processMessage(msg, topicTemplate)
	}
}

// processMessage 处理接收到的消息
// topicTemplate 为触发当前 MQTT 回调的订阅模板。
func (c *mqttClient) processMessage(msg mqtt.Message, topicTemplate string) {
	ctx := context.Background()
	payload := msg.Payload()

	// 尝试解析包装消息（用于链路追踪），只调用一次 msg.Payload()
	if wrapped, err := c.tryUnwrapPayload(payload); err == nil {
		payload = wrapped.Payload
		ctx = c.extractTraceContext(ctx, wrapped)
	}

	// 启动追踪 span
	ctx, span := c.startSpan(ctx, msg, topicTemplate)
	defer span.End()
	ctx = logx.ContextWithFields(ctx,
		logx.Field("client", c.GetClientID()),
		logx.Field("topic", msg.Topic()),
		logx.Field("topic_template", topicTemplate),
		logx.Field("payload_bytes", len(payload)),
		logx.Field("payload_size", tool.DecimalBytes(int64(len(payload)), 1)),
	)

	if len(payload) == 0 {
		logx.WithContext(ctx).Error("[mqtt] empty payload")
		return
	}

	// 分发给 handlers
	c.dispatcher.dispatch(ctx, payload, msg.Topic(), topicTemplate)
}

// tryUnwrapPayload 尝试解析包装消息
// 如果消息是 JSON 格式的 Message 结构，会提取其中的 payload
func (c *mqttClient) tryUnwrapPayload(data []byte) (*Message, error) {
	var wrapped Message
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return nil, err
	}
	if wrapped.Payload == nil {
		return nil, errors.New("invalid wrapped payload")
	}
	return &wrapped, nil
}

// extractTraceContext 从消息中提取链路追踪上下文
func (c *mqttClient) extractTraceContext(ctx context.Context, msg *Message) context.Context {
	return tracex.Extract(ctx, tracex.NewCarrier(msg.Headers))
}

// startSpan 启动追踪 span
func (c *mqttClient) startSpan(ctx context.Context, msg mqtt.Message, topicTemplate string) (context.Context, oteltrace.Span) {
	ctx, span := c.tracer.Start(ctx, "mqtt-consume",
		oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
	)
	span.SetAttributes(
		attribute.String(attrClientID, c.cfg.ClientID),
		attribute.String(attrTopicTemplate, topicTemplate),
		attribute.String(attrMsgID, fmt.Sprintf("%d", msg.MessageID())),
		attribute.Int(attrQoS, int(msg.Qos())),
		attribute.String(attrAction, "consume"),
	)
	return ctx, span
}

// PublishWithTrace 发布消息并注入 OTel 链路追踪上下文。
func (c *mqttClient) PublishWithTrace(ctx context.Context, topic string, payload []byte) (string, error) {
	msg := NewMessage(topic, payload)
	tracex.Inject(ctx, tracex.NewCarrier(msg.Headers))
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("[mqtt] marshal trace message failed: %w", err)
	}
	if err := c.Publish(ctx, topic, jsonBytes); err != nil {
		return "", err
	}
	return trace.TraceIDFromContext(ctx), nil
}

// Publish 发布消息到指定主题
func (c *mqttClient) Publish(ctx context.Context, topic string, payload []byte) error {
	if !c.client.IsConnected() {
		return errors.New("[mqtt] cannot publish: client not connected")
	}

	timeout := time.Duration(c.cfg.Timeout) * time.Millisecond
	token := c.client.Publish(topic, c.qos, false, payload)

	if !token.WaitTimeout(timeout) {
		return fmt.Errorf("[mqtt] publish to %s timeout after %v", topic, timeout)
	}
	return token.Error()
}

// Close 关闭 MQTT 连接
func (c *mqttClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		c.client.Disconnect(250)
	}
	c.handlerMgr.closeReplyHandlers()
	c.subscribed = make(map[string]struct{})
	logx.Info("[mqtt] connection closed")
}

// GetClientID 获取客户端 ID
func (c *mqttClient) GetClientID() string {
	return c.cfg.ClientID
}

// getReplyHandler returns the reply handler for a topic template, or nil.
func (c *mqttClient) getReplyHandler(topicTemplate string) ConsumeHandler {
	return c.handlerMgr.getReplyHandler(topicTemplate)
}
