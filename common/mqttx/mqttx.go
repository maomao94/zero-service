package mqttx

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/duke-git/lancet/v2/random"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/core/timex"
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
	Broker          []string `json:"broker"`
	ClientID        string   `json:"clientId,optional"`
	Username        string   `json:"username,optional"`
	Password        string   `json:"password,optional"`
	Qos             byte     `json:"qos"`
	Timeout         int      `json:"timeout,default=30"`         // 操作超时时间（秒）
	KeepAlive       int      `json:"keepalive,default=60"`       // 心跳间隔（秒）
	AutoSubscribe   bool     `json:"autoSubscribe,default=true"` // 是否自动订阅已添加处理器的主题
	SubscribeTopics []string `json:"subscribeTopics"`            // 初始需要订阅的主题
}

type Client struct {
	client     mqtt.Client
	cfg        MqttConfig
	mu         sync.RWMutex
	handlers   map[string][]ConsumeHandler // 主题 -> 处理器列表
	subscribed map[string]struct{}         // 已订阅的主题
	qos        byte
	metrics    *stat.Metrics
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
		uid, _ := random.UUIdV4()
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
	}

	// 修正QoS值
	if c.qos > 2 {
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

// 消息处理包装器：动态判断是否有处理器
func (c *Client) messageHandlerWrapper(topic string) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		ctx := logx.ContextWithFields(context.Background(), logx.Field("clientID", c.cfg.ClientID))
		startTime := timex.Now()
		defer func() {
			c.metrics.Add(stat.Task{Duration: timex.Since(startTime)})
			if r := recover(); r != nil {
				logx.WithContext(ctx).Errorf("[mqtt] handler panic for %s: %v", topic, r)
			}
		}()

		c.mu.RLock()
		handlers := c.handlers[topic]
		c.mu.RUnlock()

		if len(handlers) == 0 {
			defaultHandler{}.Consume(context.Background(), topic, msg.Payload())
			return
		}

		for _, handler := range handlers {
			if err := handler.Consume(ctx, topic, msg.Payload()); err != nil {
				logx.WithContext(ctx).Errorf("[mqtt] handler error for %s: %v", topic, err)
				c.metrics.AddDrop()
			}
		}
	}
}

// Publish 发布消息（检查连接状态）
func (c *Client) Publish(topic string, payload []byte) error {
	if !c.client.IsConnected() {
		return fmt.Errorf("[mqtt] cannot publish: client not connected")
	}

	timeout := time.Duration(c.cfg.Timeout) * time.Second
	token := c.client.Publish(topic, c.qos, false, payload)
	if !token.WaitTimeout(timeout) {
		return fmt.Errorf("[mqtt] publish to %s timeout after %v", topic, timeout)
	}
	return token.Error()
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
type defaultHandler struct {
}

func (d defaultHandler) Consume(ctx context.Context, topic string, payload []byte) error {
	logx.WithContext(ctx).Errorf("[mqtt] No handler for topic %s, add with AddHandler", topic)
	return nil
}
