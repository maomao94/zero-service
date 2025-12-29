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
	Consume(ctx context.Context, payload []byte, topic string, topicTemplate string) error
}

// ConsumeHandlerFunc 适配器，允许函数作为ConsumeHandler接口实现
type ConsumeHandlerFunc func(ctx context.Context, payload []byte, topic string, topicTemplate string) error

func (f ConsumeHandlerFunc) Consume(ctx context.Context, payload []byte, topic string, topicTemplate string) error {
	return f(ctx, payload, topic, topicTemplate)
}

// MqttConfig 定义 MQTT 客户端基础配置
type MqttConfig struct {
	Broker          []string
	ClientID        string `json:",optional"`
	Username        string `json:",optional"`
	Password        string `json:",optional"`
	Qos             byte
	Timeout         int      `json:",default=30"`   // 操作超时时间（秒）
	KeepAlive       int      `json:",default=60"`   // 心跳间隔（秒）
	AutoSubscribe   bool     `json:",default=true"` // 是否自动订阅已添加处理器的主题
	SubscribeTopics []string `json:",optional"`     // 初始需要订阅的主题
}

// Option 定义可选配置函数
type Option func(*Client)

// WithOnReady 设置首次连接成功时的回调（仅执行一次）
func WithOnReady(fn func(c *Client)) Option {
	return func(c *Client) {
		c.onReady = fn
	}
}

type Client struct {
	client     mqtt.Client
	cfg        MqttConfig
	mu         sync.RWMutex
	handlers   map[string][]ConsumeHandler // 主题 -> 处理器列表
	subscribed map[string]struct{}         // 已订阅主题
	onReady    func(c *Client)             // 第一次连接前调用
	ready      bool
	qos        byte
	metrics    *stat.Metrics
	tracer     oteltrace.Tracer
}

func MustNewClient(cfg MqttConfig, opts ...Option) *Client {
	cli, err := NewClient(cfg, opts...)
	logx.Must(err)
	proc.AddShutdownListener(func() {
		cli.Close()
	})
	return cli
}

// NewClient 创建 MQTT 客户端
func NewClient(cfg MqttConfig, opts ...Option) (*Client, error) {
	if len(cfg.Broker) == 0 {
		return nil, fmt.Errorf("[mqtt] no broker addresses provided")
	}

	if len(cfg.ClientID) == 0 {
		uid, err := random.UUIdV4()
		if err != nil {
			return nil, fmt.Errorf("[mqtt] generate clientID failed: %w", err)
		}
		cfg.ClientID = strings.ReplaceAll(uid, "-", "")
	}

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
		tracer:     otel.Tracer(trace.TraceName),
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.qos < 0 || c.qos > 2 {
		c.qos = 1
		logx.Errorf("[mqtt] Invalid QoS %d, adjusted to 1", cfg.Qos)
	}

	optsMqtt := mqtt.NewClientOptions()
	for _, broker := range cfg.Broker {
		optsMqtt.AddBroker(broker)
	}
	optsMqtt.SetClientID(cfg.ClientID).
		SetUsername(cfg.Username).
		SetPassword(cfg.Password).
		SetAutoReconnect(true).
		SetConnectTimeout(time.Duration(cfg.Timeout) * time.Second).
		SetKeepAlive(time.Duration(cfg.KeepAlive) * time.Second)

	optsMqtt.OnConnect = func(cli mqtt.Client) {
		logx.Infof("[mqtt] Connection successful, client=%s", cfg.ClientID)
		if !c.ready && c.onReady != nil {
			c.onReady(c)
			c.ready = true
		}
		if err := c.RestoreSubscriptions(); err != nil {
			logx.Errorf("[mqtt] Failed to restore subscriptions: %v", err)
		} else {
			logx.Info("[mqtt] All subscriptions restored")
		}
	}

	optsMqtt.OnConnectionLost = func(cli mqtt.Client, err error) {
		logx.Errorf("[mqtt] Connection lost: %v (auto-reconnecting)", err)
		c.mu.Lock()
		c.subscribed = make(map[string]struct{}) // 重连后需重新订阅
		c.mu.Unlock()
	}

	c.client = mqtt.NewClient(optsMqtt)
	token := c.client.Connect()
	if !token.WaitTimeout(time.Duration(cfg.Timeout) * time.Second) {
		return nil, fmt.Errorf("[mqtt] connect timeout after %ds", cfg.Timeout)
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("[mqtt] connect failed: %w", err)
	}

	return c, nil
}

// AddHandler 为主题添加处理器
func (c *Client) AddHandler(topic string, handler ConsumeHandler) error {
	if handler == nil {
		return errors.New("[mqtt] handler cannot be nil")
	}

	c.mu.Lock()
	c.handlers[topic] = append(c.handlers[topic], handler)
	needSubscribe := c.cfg.AutoSubscribe && !c.isSubscribed(topic)
	c.mu.Unlock()

	if needSubscribe && c.client.IsConnected() {
		if err := c.Subscribe(topic); err != nil {
			return err
		}
	}
	return nil
}

// AddHandlerFunc 快捷注册函数处理器
func (c *Client) AddHandlerFunc(topic string, handler func(ctx context.Context, payload []byte, topic string, topicTemplate string) error) error {
	return c.AddHandler(topic, ConsumeHandlerFunc(handler))
}

// Subscribe 手动订阅主题
func (c *Client) Subscribe(topicTemplate string) error {
	if !c.client.IsConnected() {
		return fmt.Errorf("[mqtt] cannot subscribe: client not connected")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	return c.subscribe(topicTemplate)
}

// 内部订阅实现
func (c *Client) subscribe(topicTemplate string) error {
	if c.isSubscribed(topicTemplate) {
		return nil
	}

	token := c.client.Subscribe(topicTemplate, c.qos, c.messageHandlerWrapper(topicTemplate))
	timeout := time.Duration(c.cfg.Timeout) * time.Second
	if !token.WaitTimeout(timeout) {
		return fmt.Errorf("[mqtt] subscribe to %s timeout after %v", topicTemplate, timeout)
	}
	if err := token.Error(); err != nil {
		return err
	}

	c.subscribed[topicTemplate] = struct{}{}
	logx.Infof("[mqtt] Subscribed to %s", topicTemplate)
	return nil
}

// RestoreSubscriptions 连接成功后恢复主题
func (c *Client) RestoreSubscriptions() error {
	c.mu.RLock()
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
func (c *Client) messageHandlerWrapper(topicTemplate string) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		ctx := context.Background()
		payload := msg.Payload()

		var wrapped Message
		if err := json.Unmarshal(msg.Payload(), &wrapped); err == nil && wrapped.Payload != nil {
			payload = wrapped.Payload
			carrier := NewMessageCarrier(&wrapped)
			ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
		}

		ctx, span := c.startConsumeSpan(ctx, msg, topicTemplate)
		defer span.End()
		ctx = logx.ContextWithFields(ctx, logx.Field("client", c.GetClientID()))

		startTime := timex.Now()
		defer func() {
			c.metrics.Add(stat.Task{Duration: timex.Since(startTime)})
			if r := recover(); r != nil {
				err := fmt.Errorf("handler panic: %v", r)
				logx.WithContext(ctx).Errorf("[mqtt] handler panic for %s: %v", topicTemplate, r)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}
		}()

		c.mu.RLock()
		handlers := c.handlers[topicTemplate]
		c.mu.RUnlock()

		if len(payload) == 0 {
			logx.WithContext(ctx).Errorf("[mqtt] empty payload for %s", topicTemplate)
			return
		}
		if len(handlers) == 0 {
			err := errors.New("no handler for topic")
			defaultHandler{}.Consume(ctx, payload, msg.Topic(), topicTemplate)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}

		for _, handler := range handlers {
			if err := handler.Consume(ctx, payload, msg.Topic(), topicTemplate); err != nil {
				logx.WithContext(ctx).Errorf("[mqtt] handler error for %s: %v", topicTemplate, err)
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

// Close 关闭客户端连接
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.client.Disconnect(250)
	c.subscribed = make(map[string]struct{})
	logx.Info("[mqtt] Connection closed")
}

func (c *Client) GetClientID() string {
	return c.cfg.ClientID
}

func (c *Client) isSubscribed(topic string) bool {
	_, exists := c.subscribed[topic]
	return exists
}

// defaultHandler 仅在无自定义处理器时使用
type defaultHandler struct{}

func (d defaultHandler) Consume(ctx context.Context, payload []byte, topic string, topicTemplate string) error {
	logx.WithContext(ctx).Errorf("[mqtt] No handler for topic %s, topicTemplate %s, add with AddHandler", topic, topicTemplate)
	return nil
}

// ---------------- 跟踪辅助 ----------------
func (c *Client) startConsumeSpan(ctx context.Context, msg mqtt.Message, topic string) (context.Context, oteltrace.Span) {
	msgID := fmt.Sprintf("%d", msg.MessageID())
	ctx, span := c.tracer.Start(ctx, "mqtt-consume",
		oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
	)
	span.SetAttributes(
		MqttClientIDKey.String(c.cfg.ClientID),
		MqttTopicKey.String(topic),
		MqttMessageIDKey.String(msgID),
		MqttQoSKey.Int(int(msg.Qos())),
		MqttActionKey.String("consume"),
	)
	return ctx, span
}

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
