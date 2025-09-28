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
	// Consume 处理接收到的消息
	// 返回error表示处理失败，客户端会记录错误日志
	Consume(ctx context.Context, topic string, payload []byte) error
}

// ConsumeHandlerFunc 适配器，允许函数作为ConsumeHandler接口实现
type ConsumeHandlerFunc func(ctx context.Context, topic string, payload []byte) error

// Consume 实现ConsumeHandler接口
func (f ConsumeHandlerFunc) Consume(ctx context.Context, topic string, payload []byte) error {
	return f(ctx, topic, payload)
}

type MqttConfig struct {
	Broker          []string `json:"broker"`
	ClientID        string   `json:"clientId,optional"`
	Username        string   `json:"username"`
	Password        string   `json:"password"`
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
	handlers   map[string][]ConsumeHandler // 主题 -> 处理器列表（支持多处理器）
	subscribed map[string]struct{}         // 已订阅的主题，用于去重
	qos        byte
	metrics    *stat.Metrics
}

func MustNewClient(cfg MqttConfig) *Client {
	cli, err := NewClient(cfg)
	logx.Must(err)
	return cli
}

// NewClient 创建MQTT客户端
func NewClient(cfg MqttConfig) (*Client, error) {
	if len(cfg.Broker) == 0 {
		return nil, fmt.Errorf("[mqtt] no broker addresses provided in config")
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
	for _, broker := range c.cfg.Broker {
		opts = opts.AddBroker(broker)
	}
	opts.SetClientID(cfg.ClientID).
		SetUsername(cfg.Username).
		SetPassword(cfg.Password).
		SetAutoReconnect(true).
		SetConnectTimeout(time.Duration(cfg.Timeout) * time.Second).
		SetKeepAlive(time.Duration(cfg.KeepAlive) * time.Second)

	// 连接成功回调：重新订阅所有已注册的主题
	opts.OnConnect = func(cli mqtt.Client) {
		logx.Info("[mqtt] Connected to broker, restoring subscriptions")
		c.RestoreSubscriptions()
	}

	// 连接丢失回调
	opts.OnConnectionLost = func(cli mqtt.Client, err error) {
		logx.Errorf("[mqtt] Connection lost: %v (will auto-reconnect)", err)
		// 清除已订阅标记，重连后会重新订阅
		c.mu.Lock()
		c.subscribed = make(map[string]struct{})
		c.mu.Unlock()
	}

	// 创建客户端并连接
	c.client = mqtt.NewClient(opts)
	token := c.client.Connect()
	if !token.WaitTimeout(time.Duration(cfg.Timeout) * time.Second) {
		return nil, fmt.Errorf("[mqtt] connect timeout after %d seconds", cfg.Timeout)
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("[mqtt] connect failed: %w", err)
	}

	// 初始化配置中的订阅主题（使用默认处理器）
	for _, topic := range cfg.SubscribeTopics {
		c.AddHandler(topic, defaultHandler{topic: topic})
	}

	return c, nil
}

// AddHandler 为指定主题添加消息处理器
// 如果AutoSubscribe为true，会自动订阅该主题
func (c *Client) AddHandler(topic string, handler ConsumeHandler) error {
	if handler == nil {
		return errors.New("[mqtt] handler cannot be nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 添加处理器到列表
	c.handlers[topic] = append(c.handlers[topic], handler)

	// 如果启用自动订阅，且尚未订阅，则立即订阅
	if c.cfg.AutoSubscribe {
		if _, exists := c.subscribed[topic]; !exists {
			if err := c.subscribe(topic); err != nil {
				return err
			}
		}
	}

	return nil
}

// AddHandlerFunc 为指定主题添加函数作为消息处理器（便捷方法）
func (c *Client) AddHandlerFunc(topic string, handler func(ctx context.Context, topic string, payload []byte) error) error {
	return c.AddHandler(topic, ConsumeHandlerFunc(handler))
}

// Subscribe 订阅指定主题（使用已添加的处理器）
func (c *Client) Subscribe(topic string) error {
	c.mu.RLock()
	_, hasHandler := c.handlers[topic]
	c.mu.RUnlock()

	if !hasHandler {
		return fmt.Errorf("[mqtt] no handlers for topic %s, add handlers first", topic)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	return c.subscribe(topic)
}

// 实际执行订阅操作（需在已加锁的情况下调用）
func (c *Client) subscribe(topic string) error {
	// 检查是否已订阅
	if _, exists := c.subscribed[topic]; exists {
		return nil
	}

	// 订阅主题，使用统一的消息处理包装器
	token := c.client.Subscribe(topic, c.qos, c.messageHandlerWrapper(topic))
	timeout := time.Duration(c.cfg.Timeout) * time.Second
	if !token.WaitTimeout(timeout) {
		return fmt.Errorf("[mqtt] subscribe to %s timeout after %v", topic, timeout)
	}

	if err := token.Error(); err != nil {
		return err
	}

	// 标记为已订阅
	c.subscribed[topic] = struct{}{}
	logx.Infof("[mqtt] Subscribed to topic: %s", topic)
	return nil
}

// RestoreSubscriptions 重新订阅所有已添加处理器的主题（用于重连后恢复）
func (c *Client) RestoreSubscriptions() error {
	c.mu.RLock()
	topics := make([]string, 0, len(c.handlers))
	for topic := range c.handlers {
		topics = append(topics, topic)
	}
	c.mu.RUnlock()

	var lastErr error
	for _, topic := range topics {
		if err := c.Subscribe(topic); err != nil {
			logx.Errorf("[mqtt] Failed to restore subscription to %s: %v", topic, err)
			lastErr = err
		}
	}
	return lastErr
}

// 消息处理包装器，负责调用该主题的所有处理器
func (c *Client) messageHandlerWrapper(topic string) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		startTime := timex.Now()
		defer func() {
			// 记录处理耗时
			c.metrics.Add(stat.Task{
				Duration: timex.Since(startTime),
			})
			// 捕获处理器中的panic
			if r := recover(); r != nil {
				logx.Errorf("[mqtt] Handler panic for topic %s: %v", msg.Topic(), r)
			}
		}()

		// 获取该主题的所有处理器
		c.mu.RLock()
		handlers := c.handlers[topic]
		c.mu.RUnlock()

		if len(handlers) == 0 {
			logx.Errorf("[mqtt] No handlers for topic %s", msg.Topic())
			return
		}

		// 调用所有处理器
		ctx := context.Background()
		for _, handler := range handlers {
			if err := handler.Consume(ctx, msg.Topic(), msg.Payload()); err != nil {
				logx.Errorf("[mqtt] Handler error for topic %s: %v", msg.Topic(), err)
				c.metrics.AddDrop()
			}
		}
	}
}

// Publish 发布消息
func (c *Client) Publish(topic string, payload []byte) error {
	timeout := time.Duration(c.cfg.Timeout) * time.Second
	token := c.client.Publish(topic, c.qos, false, payload)
	if !token.WaitTimeout(timeout) {
		return fmt.Errorf("[mqtt] publish to %s timeout after %v", topic, timeout)
	}
	return token.Error()
}

// Close 断开连接并清理资源
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 断开连接（等待250ms让消息发送完成）
	c.client.Disconnect(250)
	c.subscribed = make(map[string]struct{})
	logx.Info("[mqtt] Client disconnected")
}

// defaultHandler 默认处理器，用于初始订阅的主题
type defaultHandler struct {
	topic string
}

// Consume 实现ConsumeHandler接口
func (d defaultHandler) Consume(ctx context.Context, topic string, payload []byte) error {
	logx.Errorf("[mqtt] No custom handler for topic %s, please set one with AddHandler", d.topic)
	return nil
}
