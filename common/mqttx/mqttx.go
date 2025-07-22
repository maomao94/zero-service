package mqttx

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/core/timex"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/zeromicro/go-zero/core/logx"
)

type MqttConfig struct {
	Broker          []string `json:"broker"`
	ClientID        string   `json:"clientId"`
	Username        string   `json:"username"`
	Password        string   `json:"password"`
	Qos             byte     `json:"qos"`
	Timeout         int      `json:"timeout"`   // 秒
	KeepAlive       int      `json:"keepalive"` // 秒
	SubscribeTopics []string `json:"subscribeTopics"`
}

type MessageHandler func(ctx context.Context, topic string, payload []byte)

type Client struct {
	client        mqtt.Client
	broker        []string
	clientID      string
	mu            sync.Mutex
	subscriptions map[string]MessageHandler
	qos           byte
	metrics       *stat.Metrics
}

func MustNewClient(cfg MqttConfig) *Client {
	cli, err := NewClient(cfg)
	logx.Must(err)
	return cli
}

// NewClient 根据配置创建 MQTT 客户端，连接成功后自动订阅内存里已注册的所有主题
// 启动时只会注册配置中的订阅回调，但不会立即调用 Subscribe 订阅（避免重复订阅）
// 用户可以调用 Subscribe 方法动态添加订阅和回调
func NewClient(cfg MqttConfig) (*Client, error) {
	c := &Client{
		broker:        cfg.Broker,
		clientID:      cfg.ClientID,
		subscriptions: make(map[string]MessageHandler),
		qos:           cfg.Qos,
		metrics:       stat.NewMetrics(fmt.Sprintf("mqtt-%s", cfg.ClientID)),
	}

	if c.qos > 2 {
		c.qos = 1
	}
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

	opts.OnConnect = func(cli mqtt.Client) {
		logx.Info("[mqtt] Connected to broker, restoring subscriptions")
		c.mu.Lock()
		defer c.mu.Unlock()
		for topic, handler := range c.subscriptions {
			token := cli.Subscribe(topic, c.qos, func(client mqtt.Client, msg mqtt.Message) {
				startTime := timex.Now()
				defer c.metrics.Add(stat.Task{
					Duration: timex.Since(startTime),
				})
				handler(context.Background(), msg.Topic(), msg.Payload())
			})
			token.Wait()
			if token.Error() != nil {
				logx.Errorf("[mqtt] Subscribe to %s failed: %v", topic, token.Error())
			} else {
				logx.Infof("[mqtt] Subscribed to %s", topic)
			}
		}
	}

	c.client = mqtt.NewClient(opts)
	token := c.client.Connect()
	if !token.WaitTimeout(time.Duration(cfg.Timeout)*time.Second) || token.Error() != nil {
		return nil, fmt.Errorf("[mqtt] connect failed: %w", token.Error())
	}

	// 启动时注册配置中订阅主题的默认回调，不触发订阅操作，等OnConnect统一处理
	for _, topic := range cfg.SubscribeTopics {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.subscriptions[topic] = func(ctx context.Context, topic string, payload []byte) {
			logx.Infof("[mqtt] Received message on %s but no handler registered", topic)
		}
	}

	return c, nil
}

// Subscribe 注册订阅回调并立即订阅，支持覆盖回调和自动恢复
func (c *Client) Subscribe(topic string, handler MessageHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subscriptions[topic] = handler

	token := c.client.Subscribe(topic, c.qos, func(cli mqtt.Client, msg mqtt.Message) {
		startTime := timex.Now()
		defer c.metrics.Add(stat.Task{
			Duration: timex.Since(startTime),
		})
		handler(context.Background(), msg.Topic(), msg.Payload())
	})
	if !token.WaitTimeout(15 * time.Second) {
		return fmt.Errorf("[mqtt] subscribe timeout")
	}
	return token.Error()
}

// Publish 发送消息，等待确认
func (c *Client) Publish(topic string, payload []byte) error {
	token := c.client.Publish(topic, c.qos, false, payload)
	if !token.WaitTimeout(15 * time.Second) {
		return fmt.Errorf("[mqtt] publish timeout")
	}
	return token.Error()
}

// Close 断开 MQTT 连接
func (c *Client) Close() {
	c.client.Disconnect(250)
}
