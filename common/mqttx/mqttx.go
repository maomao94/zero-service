package mqttx

import (
	"context"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/core/timex"
)

type MqttConfig struct {
	Broker          []string `json:"broker"`
	ClientID        string   `json:"clientId"`
	Username        string   `json:"username"`
	Password        string   `json:"password"`
	Qos             byte     `json:"qos"`
	Timeout         int      `json:"timeout"`   // 操作超时时间（秒），用于连接、订阅、发布等
	KeepAlive       int      `json:"keepalive"` // 心跳间隔（秒）
	SubscribeTopics []string `json:"subscribeTopics"`
}

type MessageHandler func(ctx context.Context, topic string, payload []byte)

type Client struct {
	client        mqtt.Client
	cfg           MqttConfig // 保存配置，便于后续使用
	broker        []string
	clientID      string
	mu            sync.Mutex
	subscriptions map[string]MessageHandler // 存储主题与处理器的映射
	qos           byte
	metrics       *stat.Metrics
}

func MustNewClient(cfg MqttConfig) *Client {
	cli, err := NewClient(cfg)
	logx.Must(err)
	return cli
}

// NewClient 创建MQTT客户端，连接成功后自动订阅已注册的主题
func NewClient(cfg MqttConfig) (*Client, error) {
	// 初始化默认值（防止配置缺失导致异常）
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 // 默认超时30秒
	}
	if cfg.KeepAlive <= 0 {
		cfg.KeepAlive = 60 // 默认心跳60秒
	}

	c := &Client{
		broker:        cfg.Broker,
		clientID:      cfg.ClientID,
		cfg:           cfg, // 保存配置到Client
		subscriptions: make(map[string]MessageHandler),
		qos:           cfg.Qos,
		metrics:       stat.NewMetrics(fmt.Sprintf("mqtt-%s", cfg.ClientID)),
	}

	// 修正QoS值（确保在0-2范围内）
	if c.qos > 2 {
		c.qos = 1
		logx.Errorf("[mqtt] Invalid QoS %d, adjusted to 1", cfg.Qos)
	}

	// 配置MQTT客户端选项
	opts := mqtt.NewClientOptions()
	for _, broker := range c.broker {
		opts = opts.AddBroker(broker)
	}
	opts.SetClientID(cfg.ClientID).
		SetUsername(cfg.Username).
		SetPassword(cfg.Password).
		SetAutoReconnect(true).
		SetConnectTimeout(time.Duration(cfg.Timeout) * time.Second).
		SetKeepAlive(time.Duration(cfg.KeepAlive) * time.Second)

	// 连接成功回调
	opts.OnConnect = func(cli mqtt.Client) {
		logx.Info("[mqtt] Connected to broker, restoring subscriptions")
		c.mu.Lock()
		defer c.mu.Unlock()

		// 重新订阅所有已注册的主题
		for topic, handler := range c.subscriptions {
			// 订阅时使用带 metrics 的回调包装器
			token := cli.Subscribe(topic, c.qos, c.wrapHandler(handler))
			token.Wait()
			if err := token.Error(); err != nil {
				logx.Errorf("[mqtt] Failed to restore subscription to %s: %v", topic, err)
			} else {
				logx.Infof("[mqtt] Restored subscription to %s", topic)
			}
		}
	}

	// 连接丢失回调：增强可观测性
	opts.OnConnectionLost = func(cli mqtt.Client, err error) {
		logx.Errorf("[mqtt] Connection lost: %v (will auto-reconnect)", err)
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

	// 初始化配置中的订阅主题（使用默认处理器提示用户）
	for _, topic := range cfg.SubscribeTopics {
		c.mu.Lock()
		if _, exists := c.subscriptions[topic]; !exists {
			c.subscriptions[topic] = c.defaultHandler(topic)
		}
		c.mu.Unlock()
	}

	return c, nil
}

// wrapHandler 包装消息处理器，添加metrics和统一错误处理
func (c *Client) wrapHandler(handler MessageHandler) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		startTime := timex.Now()
		defer func() {
			// 记录处理耗时
			c.metrics.Add(stat.Task{
				Duration: timex.Since(startTime),
			})
			// 捕获处理器中的panic，避免客户端崩溃
			if r := recover(); r != nil {
				logx.Errorf("[mqtt] Handler panic for topic %s: %v", msg.Topic(), r)
			}
		}()

		// 调用用户提供的处理器（可考虑允许用户传入context，此处保持兼容）
		handler(context.Background(), msg.Topic(), msg.Payload())
	}
}

// defaultHandler 初始订阅的默认处理器，明确提示用户需要设置实际处理器
func (c *Client) defaultHandler(topic string) MessageHandler {
	return func(ctx context.Context, t string, payload []byte) {
		logx.Errorf("[mqtt] No custom handler for topic %s (call Subscribe to set one)", topic)
	}
}

// Subscribe 订阅主题并注册处理器（支持覆盖现有处理器）
func (c *Client) Subscribe(topic string, handler MessageHandler) error {
	if handler == nil {
		return fmt.Errorf("[mqtt] handler cannot be nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 若主题已存在，提示覆盖
	if _, exists := c.subscriptions[topic]; exists {
		logx.Errorf("[mqtt] Overwriting existing handler for topic %s", topic)
	}
	c.subscriptions[topic] = handler

	// 立即订阅（使用配置的超时时间）
	timeout := time.Duration(c.cfg.Timeout) * time.Second
	token := c.client.Subscribe(topic, c.qos, c.wrapHandler(handler))
	if !token.WaitTimeout(timeout) {
		return fmt.Errorf("[mqtt] subscribe to %s timeout after %v", topic, timeout)
	}
	return token.Error()
}

// Publish 发布消息（使用配置的超时时间）
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
	logx.Info("[mqtt] Client disconnected")
}
