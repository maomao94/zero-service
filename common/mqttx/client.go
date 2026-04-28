package mqttx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/duke-git/lancet/v2/random"
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
	attrTopic    = "mqtt.topic"
	attrClientID = "mqtt.client_id"
	attrMsgID    = "mqtt.message_id"
	attrQoS      = "mqtt.qos"
	attrAction   = "mqtt.action"
)

// Client MQTT 客户端
// 提供 MQTT 连接、订阅、发布等功能
type Client struct {
	client     mqtt.Client
	cfg        MqttConfig
	handlerMgr *handlerManager
	dispatcher *messageDispatcher
	subscribed map[string]struct{}
	ready      bool
	qos        byte
	onReady    func(*Client)
	mu         sync.RWMutex
	tracer     oteltrace.Tracer
	metrics    *stat.Metrics
}

// MustNewClient 创建 MQTT 客户端，连接失败时 panic
// 推荐在应用启动时使用，会自动注册关闭监听
func MustNewClient(cfg MqttConfig, opts ...Option) *Client {
	cli, err := NewClient(cfg, opts...)
	logx.Must(err)
	proc.AddShutdownListener(func() {
		cli.Close()
	})
	return cli
}

// NewClient 创建 MQTT 客户端
// 会自动连接 MQTT 服务器，重连由底层库处理
func NewClient(cfg MqttConfig, opts ...Option) (*Client, error) {
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

	c := &Client{
		cfg:        cfg,
		handlerMgr: newHandlerManager(),
		subscribed: make(map[string]struct{}),
		qos:        cfg.Qos,
		tracer:     otel.Tracer(trace.TraceName),
		metrics:    stat.NewMetrics(fmt.Sprintf("mqtt-%s", cfg.ClientID)),
	}
	c.dispatcher = newMessageDispatcher(c.handlerMgr, c.metrics)

	// 应用选项
	for _, opt := range opts {
		opt(c)
	}

	// 创建并连接
	client, err := c.createMqttClient()
	if err != nil {
		return nil, err
	}
	c.client = client

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

// createMqttClient 创建底层 MQTT 客户端并连接
func (c *Client) createMqttClient() (mqtt.Client, error) {
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

	client := mqtt.NewClient(optsMqtt)
	token := client.Connect()

	if !token.WaitTimeout(time.Duration(c.cfg.Timeout) * time.Millisecond) {
		return nil, fmt.Errorf("[mqtt] connect timeout after %dms", c.cfg.Timeout)
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("[mqtt] connect failed: %w", err)
	}

	return client, nil
}

// onConnect 连接成功回调
func (c *Client) onConnect(_ mqtt.Client) {
	logx.Infof("[mqtt] Connection successful, client=%s", c.cfg.ClientID)

	// 首次连接执行 onReady 回调
	if !c.ready && c.onReady != nil {
		c.onReady(c)
		c.ready = true
	}

	// 恢复订阅
	if err := c.restoreSubscriptions(); err != nil {
		logx.Errorf("[mqtt] Failed to restore subscriptions: %v", err)
	} else {
		logx.Info("[mqtt] All subscriptions restored")
	}
}

// onConnectionLost 连接丢失回调
func (c *Client) onConnectionLost(_ mqtt.Client, err error) {
	logx.Errorf("[mqtt] Connection lost: %v (auto-reconnecting)", err)
	c.mu.Lock()
	c.subscribed = make(map[string]struct{}) // 重连后需要重新订阅
	c.mu.Unlock()
}

// defaultHandler 默认消息处理器（无自定义处理器时）
func (c *Client) defaultHandler(_ mqtt.Client, msg mqtt.Message) {
	logx.Errorf("[mqtt] No handler for topic %s", msg.Topic())
}

// AddHandler 为主题添加消息处理器
// 如果 AutoSubscribe=true 且已连接，会自动订阅该主题
func (c *Client) AddHandler(topic string, handler ConsumeHandler) error {
	if handler == nil {
		return errors.New("[mqtt] handler cannot be nil")
	}

	c.handlerMgr.addHandler(topic, handler)

	c.mu.Lock()
	needSubscribe := c.cfg.AutoSubscribe && !c.isSubscribed(topic)
	c.mu.Unlock()

	if needSubscribe && c.client.IsConnected() {
		return c.subscribe(topic)
	}
	return nil
}

// AddHandlerFunc 快捷方法：用函数作为消息处理器
func (c *Client) AddHandlerFunc(topic string, fn func(context.Context, []byte, string, string) error) error {
	return c.AddHandler(topic, ConsumeHandlerFunc(fn))
}

// Subscribe 手动订阅主题
func (c *Client) Subscribe(topic string) error {
	if !c.client.IsConnected() {
		return errors.New("[mqtt] cannot subscribe: client not connected")
	}
	return c.subscribe(topic)
}

// subscribe 内部订阅实现
func (c *Client) subscribe(topic string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isSubscribed(topic) {
		return nil
	}

	token := c.client.Subscribe(topic, c.qos, c.messageHandler(topic))
	timeout := time.Duration(c.cfg.Timeout) * time.Millisecond

	if !token.WaitTimeout(timeout) {
		return fmt.Errorf("[mqtt] subscribe to %s timeout after %v", topic, timeout)
	}
	if err := token.Error(); err != nil {
		return err
	}

	c.subscribed[topic] = struct{}{}
	logx.Infof("[mqtt] Subscribed to %s", topic)
	return nil
}

// isSubscribed 检查是否已订阅
func (c *Client) isSubscribed(topic string) bool {
	_, exists := c.subscribed[topic]
	return exists
}

// restoreSubscriptions 恢复所有订阅（连接重连后）
func (c *Client) restoreSubscriptions() error {
	topics := c.handlerMgr.getAllTopics()
	topics = append(topics, c.cfg.SubscribeTopics...)
	topics = uniqueTopics(topics)

	var lastErr error
	for _, topic := range topics {
		if err := c.subscribe(topic); err != nil {
			logx.Errorf("[mqtt] failed to subscribe to %s: %v", topic, err)
			lastErr = err
		}
	}
	return lastErr
}

func uniqueTopics(topics []string) []string {
	if len(topics) < 2 {
		return topics
	}
	seen := make(map[string]struct{}, len(topics))
	result := make([]string, 0, len(topics))
	for _, topic := range topics {
		if _, ok := seen[topic]; ok {
			continue
		}
		seen[topic] = struct{}{}
		result = append(result, topic)
	}
	return result
}

// messageHandler 消息处理包装器
// topic 参数为订阅时的主题模板（可能含通配符 +/#），用于 dispatcher 精确匹配处理器
func (c *Client) messageHandler(topic string) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		c.processMessage(msg, topic)
	}
}

// processMessage 处理接收到的消息
// topicTemplate 为订阅时注册的主题模板，用于 dispatcher 查找对应处理器
func (c *Client) processMessage(msg mqtt.Message, topicTemplate string) {
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
	ctx = logx.ContextWithFields(ctx, logx.Field("client", c.GetClientID()))

	if len(payload) == 0 {
		logx.WithContext(ctx).Errorf("[mqtt] empty payload")
		return
	}

	// 分发给 handlers
	c.dispatcher.dispatch(ctx, payload, msg.Topic(), topicTemplate)
}

// tryUnwrapPayload 尝试解析包装消息
// 如果消息是 JSON 格式的 Message 结构，会提取其中的 payload
func (c *Client) tryUnwrapPayload(data []byte) (*Message, error) {
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
func (c *Client) extractTraceContext(ctx context.Context, msg *Message) context.Context {
	carrier := NewMessageCarrier(msg)
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// startSpan 启动追踪 span
func (c *Client) startSpan(ctx context.Context, msg mqtt.Message, topic string) (context.Context, oteltrace.Span) {
	ctx, span := c.tracer.Start(ctx, "mqtt-consume",
		oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
	)
	span.SetAttributes(
		attribute.String(attrClientID, c.cfg.ClientID),
		attribute.String(attrTopic, topic),
		attribute.String(attrMsgID, fmt.Sprintf("%d", msg.MessageID())),
		attribute.Int(attrQoS, int(msg.Qos())),
		attribute.String(attrAction, "consume"),
	)
	return ctx, span
}

// Publish 发布消息到指定主题
func (c *Client) Publish(ctx context.Context, topic string, payload []byte) error {
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
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		c.client.Disconnect(250)
	}
	c.subscribed = make(map[string]struct{})
	logx.Info("[mqtt] Connection closed")
}

// GetClientID 获取客户端 ID
func (c *Client) GetClientID() string {
	return c.cfg.ClientID
}
