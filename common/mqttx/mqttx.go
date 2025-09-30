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
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/core/timex"
	"github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	MqttTopicKey     = attribute.Key("mqtt.topic")
	MqttClientIDKey  = attribute.Key("mqtt.client_id")
	MqttMessageIDKey = attribute.Key("mqtt.message_id")
	MqttQoSKey       = attribute.Key("mqtt.qos")
	MqttActionKey    = attribute.Key("mqtt.action") // publish/subscribe
)

// ConsumeHandler 定义消息消费接口
type ConsumeHandler interface {
	Consume(ctx context.Context, topic string, payload []byte) error
}

// ConsumeHandlerFunc 适配器，允许函数作为ConsumeHandler接口实现
type ConsumeHandlerFunc func(ctx context.Context, topic string, payload []byte) error

func (f ConsumeHandlerFunc) Consume(ctx context.Context, topic string, payload []byte) error {
	return f(ctx, topic, payload)
}

type MqttConfig struct {
	Broker          []string
	ClientID        string `json:",optional"`
	Username        string `json:",optional"`
	Password        string `json:",optional"`
	Qos             byte
	Timeout         int      `json:",default=30"`   // 操作超时时间（秒）
	KeepAlive       int      `json:",default=60"`   // 心跳间隔（秒）
	AutoSubscribe   bool     `json:",default=true"` // 是否自动订阅已添加处理器的主题
	SubscribeTopics []string // 初始需要订阅的主题
}

type Client struct {
	client     mqtt.Client
	cfg        MqttConfig
	mu         sync.RWMutex
	handlers   map[string][]ConsumeHandler // 主题 -> 处理器列表
	subscribed map[string]struct{}         // 已订阅的主题
	qos        byte
	metrics    *stat.Metrics
	tracer     oteltrace.Tracer // 跟踪器实例
}

func MustNewClient(cfg MqttConfig) *Client {
	cli, err := NewClient(cfg)
	logx.Must(err)
	return cli
}

// NewClient 创建MQTT客户端（连接成功后通过OnConnect回调执行订阅）
func NewClient(cfg MqttConfig) (*Client, error) {
	if len(cfg.Broker) == 0 {
		return nil, fmt.Errorf("[mqtt] no broker addresses provided")
	}

	if len(cfg.ClientID) == 0 {
		uid, err := random.UUIdV4()
		if err != nil {
			return nil, fmt.Errorf("[mqtt] generate clientID failed: %w", err)
		}
		uid = strings.ReplaceAll(uid, "-", "") // 去掉所有 "-"
		cfg.ClientID = uid
	}

	// 初始化默认值
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30
	}
	if cfg.KeepAlive <= 0 {
		cfg.KeepAlive = 60
	}

	c := &Client{
		cfg:        cfg,
		handlers:   make(map[string][]ConsumeHandler),
		subscribed: make(map[string]struct{}),
		qos:        cfg.Qos,
		metrics:    stat.NewMetrics(fmt.Sprintf("mqtt-%s", cfg.ClientID)),
		tracer:     otel.Tracer(trace.TraceName), // 初始化跟踪器
	}

	// 修正QoS值
	if c.qos < 0 || c.qos > 2 {
		c.qos = 1
		logx.Errorf("[mqtt] Invalid QoS %d, adjusted to 1", cfg.Qos)
	}

	// 配置MQTT客户端选项
	opts := mqtt.NewClientOptions()
	for _, broker := range cfg.Broker {
		opts.AddBroker(broker)
	}
	opts.SetClientID(cfg.ClientID).
		SetUsername(cfg.Username).
		SetPassword(cfg.Password).
		SetAutoReconnect(true).
		SetConnectTimeout(time.Duration(cfg.Timeout) * time.Second).
		SetKeepAlive(time.Duration(cfg.KeepAlive) * time.Second)

	// 连接成功回调：所有订阅操作（初始订阅+重连恢复）都在这里执行
	opts.OnConnect = func(cli mqtt.Client) {
		logx.Info("[mqtt] Connection successful, starting subscriptions")
		// 连接成功后，恢复所有需要订阅的主题
		if err := c.RestoreSubscriptions(); err != nil {
			logx.Errorf("[mqtt] Failed to restore subscriptions: %v", err)
		} else {
			logx.Info("[mqtt] All subscriptions restored")
		}
	}

	// 连接丢失回调
	opts.OnConnectionLost = func(cli mqtt.Client, err error) {
		logx.Errorf("[mqtt] Connection lost: %v (auto-reconnecting)", err)
		c.mu.Lock()
		c.subscribed = make(map[string]struct{}) // 重连后需重新订阅
		c.mu.Unlock()
	}

	// 创建客户端并发起连接（此时还未执行订阅）
	c.client = mqtt.NewClient(opts)
	token := c.client.Connect()
	if !token.WaitTimeout(time.Duration(cfg.Timeout) * time.Second) {
		return nil, fmt.Errorf("[mqtt] connect timeout after %ds", cfg.Timeout)
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("[mqtt] connect failed: %w", err)
	}
	return c, nil
}

func (c *Client) GetClientID() string {
	return c.cfg.ClientID
}

// AddHandler 为主题添加处理器（自动订阅如果开启）
func (c *Client) AddHandler(topic string, handler ConsumeHandler) error {
	if handler == nil {
		return errors.New("[mqtt] handler cannot be nil")
	}

	c.mu.Lock()
	c.handlers[topic] = append(c.handlers[topic], handler)
	needSubscribe := c.cfg.AutoSubscribe && !c.isSubscribed(topic)
	c.mu.Unlock()

	if needSubscribe {
		// 检查当前是否已连接
		if c.client.IsConnected() {
			if err := c.Subscribe(topic); err != nil {
				return err
			}
		}
	}

	return nil
}

// AddHandlerFunc 便捷方法：添加函数作为处理器
func (c *Client) AddHandlerFunc(topic string, handler func(ctx context.Context, topic string, payload []byte) error) error {
	return c.AddHandler(topic, ConsumeHandlerFunc(handler))
}

// Subscribe 手动订阅主题（仅在已连接时执行）
func (c *Client) Subscribe(topic string) error {
	// 检查连接状态
	if !c.client.IsConnected() {
		return fmt.Errorf("[mqtt] cannot subscribe: client not connected")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	return c.subscribe(topic)
}

// 实际订阅操作（已加锁，需确保连接已建立）
func (c *Client) subscribe(topic string) error {
	if c.isSubscribed(topic) {
		return nil
	}

	token := c.client.Subscribe(topic, c.qos, c.messageHandlerWrapper(topic))
	timeout := time.Duration(c.cfg.Timeout) * time.Second
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

// RestoreSubscriptions 连接成功后恢复所有需要订阅的主题
func (c *Client) RestoreSubscriptions() error {
	c.mu.RLock()
	// 需要订阅的主题：已添加处理器的 + 配置中指定的
	topics := make(map[string]struct{})
	for topic := range c.handlers {
		topics[topic] = struct{}{}
	}
	for _, topic := range c.cfg.SubscribeTopics {
		topics[topic] = struct{}{}
	}
	c.mu.RUnlock()

	var lastErr error
	for topic := range topics {
		if err := c.Subscribe(topic); err != nil {
			logx.Errorf("[mqtt] failed to subscribe to %s: %v", topic, err)
			lastErr = err
		}
	}
	return lastErr
}

// 消息处理包装器
func (c *Client) messageHandlerWrapper(topic string) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		// 默认上下文
		ctx := context.Background()

		// --- Step 1: 尝试解析为包装消息 ---
		var wrapped Message
		if err := json.Unmarshal(msg.Payload(), &wrapped); err == nil && wrapped.Payload != nil {
			// 包装过的消息，提取 trace
			carrier := NewMessageCarrier(&wrapped)
			ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
		}

		// --- Step 2: 创建消费 span ---
		ctx, span := c.startConsumeSpan(ctx, msg, topic)
		defer span.End()
		ctx = logx.ContextWithFields(ctx, logx.Field("client", c.GetClientID()))

		// --- Step 3: 处理时间统计和 panic 捕获 ---
		startTime := timex.Now()
		defer func() {
			c.metrics.Add(stat.Task{Duration: timex.Since(startTime)})
			if r := recover(); r != nil {
				err := fmt.Errorf("handler panic: %v", r)
				logx.WithContext(ctx).Errorf("[mqtt] handler panic for %s: %v", topic, r)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}
		}()

		// --- Step 4: 分发给 handler ---
		c.mu.RLock()
		handlers := c.handlers[topic]
		c.mu.RUnlock()

		if len(handlers) == 0 {
			err := errors.New("no handler for topic")
			defaultHandler{}.Consume(ctx, topic, msg.Payload()) // 仍然传原始 payload
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}

		for _, handler := range handlers {
			if err := handler.Consume(ctx, topic, msg.Payload()); err != nil {
				logx.WithContext(ctx).Errorf("[mqtt] handler error for %s: %v", topic, err)
			}
		}
	}
}

// Publish 发布消息
func (c *Client) Publish(ctx context.Context, topic string, payload []byte) error {
	if !c.client.IsConnected() {
		return fmt.Errorf("[mqtt] cannot publish: client not connected")
	}

	_, span := c.startPublishSpan(ctx, topic)
	defer span.End()

	timeout := time.Duration(c.cfg.Timeout) * time.Second
	token := c.client.Publish(topic, c.qos, false, payload)
	if !token.WaitTimeout(timeout) {
		err := fmt.Errorf("[mqtt] publish to %s timeout after %v", topic, timeout)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if err := token.Error(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

// Close 关闭客户端
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.client.Disconnect(250)
	c.subscribed = make(map[string]struct{})
	logx.Info("[mqtt] Client closed")
}

// 判断主题是否已订阅（已加锁时使用）
func (c *Client) isSubscribed(topic string) bool {
	_, exists := c.subscribed[topic]
	return exists
}

// defaultHandler 仅在无自定义处理器时使用
type defaultHandler struct{}

func (d defaultHandler) Consume(ctx context.Context, topic string, payload []byte) error {
	logx.WithContext(ctx).Errorf("[mqtt] No handler for topic %s, add with AddHandler", topic)
	return nil
}

// ---------------- 跟踪辅助方法 ----------------

// startConsumeSpan 创建消息消费的span
func (c *Client) startConsumeSpan(ctx context.Context, msg mqtt.Message, topic string) (context.Context, oteltrace.Span) {
	// 从消息中提取或生成消息ID（Paho的Message有MessageID()方法）
	msgID := fmt.Sprintf("%d", msg.MessageID())

	ctx, span := c.tracer.Start(ctx, "mqtt-consume",
		oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
	)
	// 添加关键属性，便于追踪和排查问题
	span.SetAttributes(
		MqttClientIDKey.String(c.cfg.ClientID),
		MqttTopicKey.String(topic),
		MqttMessageIDKey.String(msgID),
		MqttQoSKey.Int(int(msg.Qos())),
		MqttActionKey.String("consume"),
	)
	return ctx, span
}

// startPublishSpan 创建消息发布的span
func (c *Client) startPublishSpan(ctx context.Context, topic string) (context.Context, oteltrace.Span) {
	ctx, span := c.tracer.Start(ctx, "mqtt-publish",
		oteltrace.WithSpanKind(oteltrace.SpanKindProducer),
	)
	span.SetAttributes(
		MqttClientIDKey.String(c.cfg.ClientID),
		MqttTopicKey.String(topic),
		MqttQoSKey.Int(int(c.qos)),
		MqttActionKey.String("publish"),
	)
	return ctx, span
}
